package main

import (
	"fmt"
	"testing"

	"github.com/garyburd/redigo/redis"
)

// TODO better fake data
func makeFakeData(count int) []string {
	data := make([]string, 0, count)
	for i := 0; i < count; i++ {
		data = append(data, fmt.Sprintf("fake data %d", i))
	}
	return data
}

func doWrites(b *testing.B, writer RedisWriter, mapkey string, data []string) {
	for _, datum := range data {
		if err := writer.Send("HSET", mapkey, datum, datum); err != nil {
			b.Fatal(err)
		}
	}
	writer.Flush()
}

func benchmarkRedisWriter(b *testing.B, flushInterval int) {
	redisConn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		b.Fatal(err)
	}
	defer redisConn.Close()
	writer := &redisWriter{conn: redisConn, flushInterval: flushInterval}
	rmap := "moredis:map:1"
	fakeData := makeFakeData(100000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		doWrites(b, writer, rmap, fakeData)
		b.StopTimer()
		redisConn.Do("FLUSHDB")
	}
}

func BenchmarkRedisWriter1000(b *testing.B) {
	benchmarkRedisWriter(b, 1000)
}

func BenchmarkRedisWriter100(b *testing.B) {
	benchmarkRedisWriter(b, 100)
}

func BenchmarkRedisWriter10(b *testing.B) {
	benchmarkRedisWriter(b, 10)
}
