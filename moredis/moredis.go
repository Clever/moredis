package moredis

import (
	"encoding/json"
	"fmt"

	"github.com/Clever/moredis/logger"
	"github.com/garyburd/redigo/redis"
	"gopkg.in/mgo.v2/bson"
)

// Params holds the params the user passes in for substitution into config templates.
type Params map[string]string

// Set parses a Param object from a string into the method receiver.
func (p *Params) Set(value string) error {
	if err := json.Unmarshal([]byte(value), p); err != nil {
		return err
	}
	return nil
}

// String gives a string representation of a Params object
func (p *Params) String() string {
	return fmt.Sprintf("%v", *p)
}

// Bson makes a copy of the Params object in a bson map for
// interaction with mgo.
func (p *Params) Bson() bson.M {
	ret := bson.M{}
	for key, val := range *p {
		ret[key] = val
	}
	return ret
}

// BuildCache builds a redis cache according to the passed in config.
func BuildCache(cacheConfig CacheConfig, params Params, redisURL string, mongoURL string, mongoDBName string) {
	logger.Info("Populating cache.", logger.M{"cache": cacheConfig.Name})

	// set up mongo/redis connections
	mongoURL = FlagEnvOrDefault(mongoURL, "MONGO_URL", DefaultMongoURL)
	mongoDBName = FlagEnvOrDefault(mongoDBName, "MONGO_DB", DefaultMongoDB)
	redisURL = FlagEnvOrDefault(redisURL, "REDIS_URL", DefaultRedisURL)
	mongoDb, redisConn, err := SetupDbs(mongoURL, mongoDBName, redisURL)
	if err != nil {
		logger.Error("Failed to connect to dbs", err)
	}
	defer mongoDb.Session.Close()
	defer redisConn.Close()
	redisWriter := NewRedisWriter(redisConn)

	for _, collection := range cacheConfig.Collections {
		query, err := ParseQuery(collection.Query, params)
		if err != nil {
			logger.Error("Failed to parse query", err)
		}

		if err := SetRedisHashKeys(redisConn, &collection); err != nil {
			logger.Error("Error setting up redis map keys", err)
		}

		logger.Info("Processing query for collection", logger.M{"query": query, "collection": collection.Collection})
		iter := mongoDb.C(collection.Collection).Find(query).Iter()
		if err := ProcessQuery(redisWriter, iter, collection.Maps); err != nil {
			logger.Error("Error processing query", err)
		}
		redisWriter.Flush()

		for _, rmap := range collection.Maps {
			err := UpdateRedisMapReference(redisConn, params, rmap)
			if err != nil {
				logger.Error("Failed to update map reference", err)
			}
		}
	}
	logger.Info("Completed populating cache", logger.M{"cache": cacheConfig.Name})
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
		}
		processed++
	}
	logger.Info("Processed all documents for query", logger.M{"processed": processed})
	return nil
}

// SetRedisHashKeys determines the correct keys to use for the redis hashes that
// will be created to store the mapped values.  These keys are generated in an atomic
// fashion and will not interfere with any other running instances of moredis
func SetRedisHashKeys(conn redis.Conn, collection *CollectionConfig) error {
	for ix := range collection.Maps {
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
func UpdateRedisMapReference(conn redis.Conn, params Params, mapConfig MapConfig) error {
	mapName, err := ApplyTemplate(mapConfig.Name, params.Bson())
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
