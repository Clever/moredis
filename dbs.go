package main

import (
	"github.com/garyburd/redigo/redis"
	"gopkg.in/mgo.v2"

	"github.com/Clever/moredis/logger"
)

// Default database connection parameters
const (
	DefaultMongoURL = "localhost:27017"
	DefaultMongoDB  = ""
	DefaultRedisURL = "localhost:6379"
)

// SetupDbs takes connection parameters for redis and mongo and returns active sessions.
// The caller is responsible for closing the returned connections.
func SetupDbs(mongoURL, mongoDBName, redisURL string) (*mgo.Database, redis.Conn, error) {
	mongoSession, err := mgo.Dial(mongoURL)
	if err != nil {
		return nil, nil, err
	}
	mongoDB := mongoSession.DB(mongoDBName)
	logger.Info("Connected to mongo", logger.M{"mongo_url": mongoURL, "mongo_db": mongoDBName})

	redisConn, err := redis.Dial("tcp", redisURL)
	if err != nil {
		return nil, nil, err
	}
	logger.Info("Connected to redis", logger.M{"redis_url": redisURL})
	return mongoDB, redisConn, nil
}

// MongoIter defines an interface that must be met by types we use as mongo iterators.
// The main purpose of breaking this out into an interface is for ease of mocking in tests.
type MongoIter interface {
	Next(result interface{}) bool
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
		if err := r.conn.Flush(); err != nil {
			return err
		}
		r.currentCount = 0
	}
	return nil

}

// Flush triggers a flush on the underlying redis connection.
func (r *redisWriter) Flush() error {
	return r.conn.Flush()
}
