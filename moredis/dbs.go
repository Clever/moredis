package moredis

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Clever/moredis/logger"
	"github.com/garyburd/redigo/redis"
	"gopkg.in/mgo.v2"
)

// SetupDbs takes connection parameters for redis and mongo and returns active sessions.
// The caller is responsible for closing the returned connections.
func SetupDbs(mongoURL, redisURL string) (*mgo.Database, redis.Conn, error) {
	mongoSession, err := mgo.Dial(mongoURL)
	if err != nil {
		return nil, nil, err
	}
	// use 'monotonic' consistency mode.  Since we only do reads, this doesn't have an actual effect
	// other than letting us read from secondaries if the connection string has the ?connect=direct param.
	mongoSession.SetMode(mgo.Monotonic, false)

	// empty db string uses the db from the connection url
	mongoDB := mongoSession.DB("")
	logger.Info("Connected to mongo", logger.M{"mongo_url": mongoURL})

	redisURL, err = resolveRedis(redisURL)
	if err != nil {
		return nil, nil, err
	}

	redisConn, err := redis.DialTimeout("tcp", redisURL, 15*time.Second, 10*time.Second, 10*time.Second)
	if err != nil {
		return nil, nil, err
	}
	logger.Info("Connected to redis", logger.M{"redis_url": redisURL})
	return mongoDB, redisConn, nil
}

// resolveRedis takes in a redis address and checks for a sentinel:// prefix to resolve. If one is present, it uses
// sentinel to resolve the master and returns that. Otherwise, it returns the original address.
func resolveRedis(address string) (string, error) {
	if !strings.HasPrefix(address, "sentinel://") {
		return address, nil
	}

	match := regexp.MustCompile(`sentinel://([^/]+)/(.*)`)
	addressParts := match.FindAllStringSubmatch(address, -1)

	if len(addressParts) < 1 || len(addressParts[0]) < 3 {
		return "", fmt.Errorf("Failed to parse sentinel address %v", address)
	}
	for _, url := range strings.Split(addressParts[0][1], ",") {
		redisConn, err := redis.Dial("tcp", url)
		if err != nil {
			continue
		}
		defer redisConn.Close()
		master, err := redis.Strings(redisConn.Do("SENTINEL", "get-master-addr-by-name", addressParts[0][2]))
		if err == nil {
			return master[0] + ":" + master[1], nil
		}
	}

	return "", fmt.Errorf("Failed to find master for sentinel address %v", address)
}

// MongoIter defines an interface that must be met by types we use as mongo iterators.
// The main purpose of breaking this out into an interface is for ease of mocking in tests.
type MongoIter interface {
	Next(result interface{}) bool
	Err() error
	Close() error
}

// RedisWriter is an interface for types that can write to redis using send/flush (pipelined operations)
// The main purpose of breaking this out into an interface is for ease of mocking in tests.
type RedisWriter interface {
	Send(cmd string, args ...interface{}) error
	Flush() error
}

type redisWriter struct {
	conn          redis.Conn
	flushInterval int
	currentCount  int
}

// NewRedisWriter creates a new RedisWriter.  We wrap redis.Conn here so that we can specify how many
// documents we want to allow buffered before flushing automatically.
func NewRedisWriter(conn redis.Conn) RedisWriter {
	writer := &redisWriter{
		conn:          conn,
		flushInterval: 100,
	}
	return writer
}

// Send uses the same interface as redis.Conn.Send().  The difference is that RedisWriter's Send
// method takes care of automatically flushing after flushInterval amount of documents.
func (r *redisWriter) Send(cmd string, args ...interface{}) error {
	if err := r.conn.Send(cmd, args...); err != nil {
		return err
	}
	r.currentCount++
	if r.currentCount >= r.flushInterval {
		if err := r.Flush(); err != nil {
			return err
		}
		r.currentCount = 0
		// Do a ping and wait for the reply to ensure that there is no data waiting to be received.
		if _, err := r.conn.Do("PING"); err != nil {
			return err
		}
	}
	return nil

}

// Flush triggers a flush on the underlying redis connection.
func (r *redisWriter) Flush() error {
	return r.conn.Flush()
}
