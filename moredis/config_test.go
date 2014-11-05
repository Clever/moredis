package moredis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var sampleConfig = Config{
	Caches: []CacheConfig{
		{
			Name:        "cache1",
			Collections: []CollectionConfig{},
		},
		{
			Name:        "cache2",
			Collections: []CollectionConfig{},
		},
	},
}

func TestGetCache(t *testing.T) {
	cacheOne, err := sampleConfig.GetCache("cache1")
	assert.Nil(t, err)
	assert.Equal(t, "cache1", cacheOne.Name)

	cacheTwo, err := sampleConfig.GetCache("cache2")
	assert.Nil(t, err)
	assert.Equal(t, "cache2", cacheTwo.Name)

	_, err = sampleConfig.GetCache("cache3")
	assert.Error(t, err)
}
