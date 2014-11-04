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

// Params holds the params the user passes in for substitution into config templates.
type Params map[string]string

func (p *Params) Set(value string) error {
	if err := json.Unmarshal([]byte(value), p); err != nil {
		return err
	}
	return nil
}

func (p *Params) String() string {
	return fmt.Sprintf("%v", *p)
}

func (p *Params) Bson() bson.M {
	ret := bson.M{}
	for key, val := range *p {
		ret[key] = val
	}
	return ret
}

var (
	redisUrl    string
	mongoUrl    string
	mongoDbName string
	cache       string
	params      Params
)

func init() {
	// Usage strings in PrintUsage
	flag.StringVar(&redisUrl, "redis_url", "", "")
	flag.StringVar(&redisUrl, "r", "", "")
	flag.StringVar(&mongoUrl, "mongo_url", "", "")
	flag.StringVar(&mongoUrl, "m", "", "")
	flag.StringVar(&mongoDbName, "mongo_db", "", "")
	flag.StringVar(&mongoDbName, "d", "", "")
	flag.StringVar(&cache, "cache", "", "")
	flag.StringVar(&cache, "c", "", "")
	flag.Var(&params, "params", "")
	flag.Var(&params, "p", "")
}

func main() {
	flag.Usage = PrintUsage
	flag.Parse()

	// cache is the only required parameter
	if cache == "" {
		flag.Usage()
		os.Exit(1)
	}

	config, err := LoadConfig("./config.yml")
	if err != nil {
		logger.Error("Error loading config.", err)
	}

	cacheConfig, err := config.GetCache(cache)
	if err != nil {
		logger.Error("Cache not found in config.", err)
	}

	logger.Info("Populating cache.", logger.M{"cache": cache})

	// set up mongo/redis connections
	mongoUrl := FlagEnvOrDefault(mongoUrl, "MONGO_URL", DEFAULT_MONGO_URL)
	mongoDbName := FlagEnvOrDefault(mongoDbName, "MONGO_DB", DEFAULT_MONGO_DB)
	redisUrl := FlagEnvOrDefault(redisUrl, "REDIS_URL", DEFAULT_REDIS_URL)
	mongoDb, redisConn, err := SetupDbs(mongoUrl, mongoDbName, redisUrl)
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
			err := UpdateRedisMapReference(redisConn, params, rmap)
			if err != nil {
				logger.Error("Failed to update map reference", err)
			}
		}
	}
	logger.Info("Completed populating cache", logger.M{"cache": cache})
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

// PrintUsage is used to replace flag.Usage, which is pretty terrible.
func PrintUsage() {
	var usage = `Usage of ./moredis:
  -c, -cache        Which cache to populate (REQUIRED)
  -d, -mongo_db     MongoDB Database, can also be set via the MONGO_DB environment variable
  -m, -mongo_url    MongoDB URL, can also be set via the MONGO_URL environment variable
  -p, -params       JSON object with params used for substitution into queries and collection names in config.yml
  -r, -redis_url    Redis URL, can also be set via the REDIS_URL environment variable
  -h, -help         Print this usage message.
`
	fmt.Fprint(os.Stderr, usage)
}
