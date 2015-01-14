package moredis

import (
	"errors"
	"reflect"
	"testing"

	"github.com/garyburd/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

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

func (m *MockIter) Err() error {
	return nil
}

func TestProcessQuery(t *testing.T) {
	iter := NewMockIter([]bson.M{{"test": "1", "val": "expected"}})

	collection := CollectionConfig{
		Maps: []MapConfig{
			{
				Key:     "{{.test}}",
				Value:   "{{.val}}",
				HashKey: "moredis:maps:1",
			},
		},
	}
	redigomock.Clear()
	redigomock.Command("HSET", "moredis:maps:1", "1", "expected").Expect("ok")
	writer := NewRedisWriter(redigomock.NewConn())
	err := ParseTemplates(&collection)
	assert.Nil(t, err)
	err = ProcessQuery(writer, iter, collection.Maps)
	assert.Nil(t, err)
}

func TestUpdateRedisMapReferenceNoOldMap(t *testing.T) {
	// should work with no previous map
	redigomock.Clear()
	redigomock.Command("GETSET", "map", "map:1").ExpectError(redis.ErrNil)
	err := UpdateRedisMapReference(redigomock.NewConn(),
		Params{},
		MapConfig{
			Name:    "map",
			HashKey: "map:1",
		},
	)

	assert.Nil(t, err)
}

func TestUpdateRedisMapReferenceWithOldMap(t *testing.T) {
	// should work with a previous map
	redigomock.Clear()
	redigomock.Command("GETSET", "map", "map:2").Expect("map:1")
	redigomock.Command("DEL", "map:1").Expect("ok")
	err := UpdateRedisMapReference(redigomock.NewConn(),
		Params{},
		MapConfig{
			Name:    "map",
			HashKey: "map:2",
		},
	)
	assert.Nil(t, err)
}

func TestUpdateRedisMapReferenceRedisErrors(t *testing.T) {
	// should return error if redis does
	redigomock.Clear()
	redigomock.Command("GETSET", "map", "map:1").ExpectError(errors.New("redis error"))
	err := UpdateRedisMapReference(redigomock.NewConn(),
		Params{},
		MapConfig{
			Name:    "map",
			HashKey: "map:1",
		},
	)
	assert.EqualError(t, err, "redis error")

	redigomock.Clear()
	redigomock.Command("GETSET", "map", "map:1").Expect("map:0")
	redigomock.Command("DEL", "map:0").ExpectError(errors.New("redis error"))
	err = UpdateRedisMapReference(redigomock.NewConn(),
		Params{},
		MapConfig{
			Name:    "map",
			HashKey: "map:1",
		},
	)
	assert.EqualError(t, err, "redis error")

}

func TestSetRedisHashKeys(t *testing.T) {
	redigomock.Clear()
	redigomock.Command("INCR", "moredis:mapindexcounter").Expect(int64(1))

	collectionConfig := CollectionConfig{Maps: []MapConfig{MapConfig{}}}
	err := SetRedisHashKeys(redigomock.NewConn(), &collectionConfig)
	assert.Nil(t, err)

	assert.Equal(t, collectionConfig.Maps[0].HashKey, "moredis:maps:1")
}

func TestSetRedisHashKeysRedisError(t *testing.T) {
	redigomock.Clear()
	redigomock.Command("INCR", "moredis:mapindexcounter").ExpectError(errors.New("redis error"))
	collectionConfig := CollectionConfig{Maps: []MapConfig{MapConfig{}}}
	err := SetRedisHashKeys(redigomock.NewConn(), &collectionConfig)
	assert.EqualError(t, err, "redis error")
}
