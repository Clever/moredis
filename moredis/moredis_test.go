package moredis

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

// MockRedisWriter is a type that can be used in place of redisWriter for tests.  Instead of sending
// commands to redis, it simply keeps them in a Commands slice.
type MockRedisWriter struct {
	Commands []string
}

func NewMockRedisWriter() *MockRedisWriter {
	return &MockRedisWriter{Commands: make([]string, 0)}
}

func (m *MockRedisWriter) Send(cmd string, args ...interface{}) error {
	for _, arg := range args {
		argStr, ok := arg.(string)
		if !ok {
			return errors.New("non string arg to Send")
		}
		cmd += " " + argStr
	}
	m.Commands = append(m.Commands, cmd)
	return nil
}

func (m *MockRedisWriter) Flush() error {
	return nil
}

// MockIter is an iterator that can be used to mock mgo's Iter type.  You give it a
// list of records to return and it will return them in order.
type MockIter struct {
	Records []bson.M
	current int
}

func NewMockIter(records []bson.M) *MockIter {
	return &MockIter{records, 0}
}

func (m *MockIter) Next(result interface{}) bool {
	if m.current >= len(m.Records) {
		return false
	}
	rv := reflect.ValueOf(result)
	p := reflect.Indirect(rv)
	p.Set(reflect.ValueOf(m.Records[m.current]))
	m.current++
	return true
}

func (m *MockIter) Close() error {
	return nil
}

func TestProcessQuery(t *testing.T) {
	iter := NewMockIter([]bson.M{{"test": "1", "val": "expected"}, {"test": "2", "val": "expected"}})

	collection := CollectionConfig{
		Maps: []MapConfig{
			{
				Key:     "{{.test}}",
				Value:   "{{.val}}",
				HashKey: "moredis:maps:1",
			},
		},
	}
	writer := NewMockRedisWriter()
	err := ParseTemplates(&collection)
	assert.Nil(t, err)
	err = ProcessQuery(writer, iter, collection.Maps)
	assert.Nil(t, err)
	assert.Equal(t, []string{"HSET moredis:maps:1 1 expected", "HSET moredis:maps:1 2 expected"}, writer.Commands)
}
