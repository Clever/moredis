package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/Clever/moredis/logger"
	"github.com/garyburd/redigo/redis"
	"gopkg.in/mgo.v2/bson"
)

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		logger.Critical("Got invalid args", logger.M{"NArgs": flag.NArg(), "Args": flag.Args()})
		PrintUsage()
		os.Exit(1)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(flag.Arg(0)), &payload); err != nil {
		logger.Error("Couldn't unmarshal payload", err)
	}

	cacheToPopulate, ok := payload["cache"].(string)
	if !ok {
		logger.Critical("Missing 'cache' in payload", logger.M{"payload": payload})
		os.Exit(1)
	}

	config, err := LoadConfig("./config.yml")
	if err != nil {
		logger.Error("Error loading config.", err)
	}

	cache, err := config.GetCache(cacheToPopulate)
	if err != nil {
		logger.Error("Cache not found in config.", err)
	}

	logger.Info("Populating cache.", logger.M{"cache": cacheToPopulate})

	// set up mongo/redis connections
	mongoUrl := PayloadOrEnv(payload, "mongo_url", DEFAULT_MONGO_URL)
	mongoDbName := PayloadOrEnv(payload, "mongo_db", "")
	redisUrl := PayloadOrEnv(payload, "redis_url", DEFAULT_REDIS_URL)
	mongoDb, redisConn, err := SetupDbs(mongoUrl, mongoDbName, redisUrl)
	if err != nil {
		logger.Error("Failed to connect to dbs", err)
	}
	defer mongoDb.Session.Close()
	defer redisConn.Close()
	redisWriter := NewRedisWriter(redisConn)

	for _, collection := range cache.Collections {
		query, err := ParseQuery(collection.Query, payload)
		if err != nil {
			logger.Error("Failed to parse query", err)
		}

		if err = SetRedisHashKeys(redisConn, &collection); err != nil {
			logger.Error("Error setting up redis map keys", err)
		}

		logger.Info("Processing query for collection", logger.M{"query": query, "collection": collection.Collection})
		iter := mongoDb.C(collection.Collection).Find(query).Iter()
		if err = ProcessQuery(redisWriter, iter, collection.Maps); err != nil {
			logger.Error("Error processing query", err)
		}
		redisWriter.Flush()

		for _, rmap := range collection.Maps {
			err := UpdateRedisMapReference(redisConn, payload, rmap)
			if err != nil {
				logger.Error("Failed to update map reference", err)
			}
		}
	}
	logger.Info("Completed populating cache", logger.M{"cache": cacheToPopulate})
}

// ProcessQuery iterates through all of the documents contained within iter, and maps
// keys to values in a redis hash according to your mapping config.
func ProcessQuery(writer RedisWriter, iter MongoIter, maps []MapConfig) error {
	processed := 0
	var result bson.M
	for iter.Next(&result) {
		for _, rmap := range maps {
			key, err := ApplyTemplate(rmap.Key, result)
			if err != nil {
				return err
			}
			if key == "" || key == "<no value>" {
				continue
			}

			val, err := ApplyTemplate(rmap.Value, result)
			if err != nil {
				return err
			}
			writer.Send("HSET", rmap.HashKey, key, val)
			processed++
		}
	}
	logger.Info("Processed all documents for query", logger.M{"processed": processed})
	return nil
}

// SetRedisHashKeys determines the correct keys to use for the redis hashes that
// will be created to store the mapped values.  These keys are generated in an atomic
// fashion and will not interfere with any other running instances of moredis
func SetRedisHashKeys(conn redis.Conn, collection *CollectionConfig) error {
	for ix, _ := range collection.Maps {
		tempKey, err := redis.Int64(conn.Do("INCR", "moredis:mapindexcounter"))
		if err != nil {
			return err
		}
		collection.Maps[ix].HashKey = fmt.Sprintf("moredis:maps:%d", tempKey)
	}
	return nil
}

// UpdateRedisMapReference updates the map specified in redis to point to the newly populated hashes,
// then deletes the previously referenced hash.  The hash reference is updated atomically.
func UpdateRedisMapReference(conn redis.Conn, payload map[string]interface{}, mapConfig MapConfig) error {
	mapName, err := ApplyTemplate(mapConfig.Name, payload)
	if err != nil {
		return err
	}
	oldMap, err := redis.String(conn.Do("GETSET", mapName, mapConfig.HashKey))
	logger.Info("Updating map reference", logger.M{"map": mapName, "oldref": oldMap, "newref": mapConfig.HashKey})
	if err == redis.ErrNil {
		// no old map, just return
		return nil
	}
	if err != nil {
		return err
	}

	logger.Info("Deleting old referenced map", logger.M{"map": oldMap})
	conn.Do("DEL", oldMap)
	return nil
}

// PrintUsage prints the usage for moredis.
func PrintUsage() {
	fmt.Println("Usage: moredis <payload>")
}
