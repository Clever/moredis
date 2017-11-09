package moredis

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/Clever/moredis/logger"
	"github.com/garyburd/redigo/redis"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Params holds the params the user passes in for substitution into config templates.
type Params map[string]string

// Set parses a Param object from a string into the method receiver.
func (p *Params) Set(value string) error {
	return json.Unmarshal([]byte(value), p)
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
func BuildCache(cacheConfig Config, params Params, redisURL string, mongoURL string) error {
	logger.Info("Populating cache.", logger.M{"cache": cacheConfig.Name})

	// set up mongo/redis connections
	mongoDb, redisConn, err := SetupDbs(mongoURL, redisURL)
	if err != nil {
		logger.Error("Failed to connect to dbs", err)
		return err
	}
	defer mongoDb.Session.Close()
	defer redisConn.Close()

	return processCollections(cacheConfig, params, mongoDb, redisConn)
}

func processCollections(cacheConfig Config, params Params, mongoDb *mgo.Database, redisConn redis.Conn) error {
	redisWriter := NewRedisWriter(redisConn)
	for _, collection := range cacheConfig.Collections {
		query, err := ParseTemplatedJSON(collection.Query, params)
		if err != nil {
			logger.Error("Failed to parse query", err)
			return err
		}

		var iter MongoIter
		var projection map[string]interface{}
		if collection.Projection != "" {
			var err error
			projection, err = ParseTemplatedJSON(collection.Projection, params)
			if err != nil {
				logger.Error("Error applying projection template", err)
			}
			iter = mongoDb.C(collection.Collection).Find(query).Select(projection).Iter()
		} else {
			iter = mongoDb.C(collection.Collection).Find(query).Iter()
		}

		if err := SetRedisHashKeys(redisConn, &collection); err != nil {
			logger.Error("Error setting up redis map keys", err)
			return err
		}

		if err := ParseTemplates(&collection); err != nil {
			logger.Error("Error parsing templates", err)
			return err
		}

		logger.Info("Processing query for collection", logger.M{
			"query":      query,
			"collection": collection.Collection,
			"projection": projection,
		})
		if err := ProcessQuery(redisWriter, iter, collection.Maps); err != nil {
			logger.Error("Error processing query", err)
			return err
		}
		if err := redisWriter.Flush(); err != nil {
			logger.Error("Error flushing redis conn", err)
			return err
		}

		for _, rmap := range collection.Maps {
			if err := UpdateRedisMapReference(redisConn, params, rmap); err != nil {
				logger.Error("Failed to update map reference", err)
				return err
			}
		}
	}
	logger.Info("Completed populating cache", logger.M{"cache": cacheConfig.Name})
	return nil
}

// ProcessQuery iterates through all of the documents contained within iter, and maps
// keys to values in a redis hash according to your mapping config.
func ProcessQuery(writer RedisWriter, iter MongoIter, maps []MapConfig) error {
	processed := 0
	var result bson.M
	var b bytes.Buffer
	for iter.Next(&result) {
		for _, rmap := range maps {
			if err := rmap.KeyTemplate.Execute(&b, result); err != nil {
				logger.Error("Could not execute key template", err)
				return err
			}
			key := b.String()
			b.Reset()

			if key == "" || key == "<no value>" {
				continue
			}

			if err := rmap.ValueTemplate.Execute(&b, result); err != nil {
				logger.Error("Could not execute value template", err)
				return err
			}
			val := b.String()
			b.Reset()

			if err := writer.Send("HSET", rmap.HashKey, key, val); err != nil {
				logger.Error("Could not send HSET", err)
				return err
			}
		}
		processed++
	}
	if err := iter.Err(); err != nil {
		logger.Error("Iteration error", err)
		return err
	}
	if err := iter.Close(); err != nil {
		logger.Error("Iter.Close() error", err)
		return err
	}
	if err := writer.Flush(); err != nil {
		logger.Error("Error flushing", err)
		return err
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
	if _, err := conn.Do("DEL", oldMap); err != nil {
		return err
	}
	return nil
}
