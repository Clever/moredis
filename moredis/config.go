package moredis

import (
	"errors"
	"io/ioutil"

	"gopkg.in/v2/yaml"
)

// Config holds the config loaded from config.yml
type Config struct {
	Caches []CacheConfig `yaml:"caches"`
}

// CacheConfig is the config for a specific cache
type CacheConfig struct {
	Name        string             `yaml:"name"`
	Collections []CollectionConfig `yaml:"collections"`
}

// CollectionConfig is the config for a specific collection
type CollectionConfig struct {
	Collection string      `yaml:"collection"`
	Query      string      `yaml:"query"`
	Maps       []MapConfig `yaml:"maps"`
}

// MapConfig is the config for a specific map.
type MapConfig struct {
	Name    string `yaml:"name"`
	Key     string `yaml:"key"`
	Value   string `yaml:"val"`
	HashKey string
}

// LoadConfig takes a path to a config yaml file and loads it into the appropriate structs.
func LoadConfig(path string) (Config, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var conf Config
	if err := yaml.Unmarshal(raw, &conf); err != nil {
		return Config{}, err
	}

	return conf, nil
}

// GetCache returns the config for a specific cache inside the moredis config.
func (c Config) GetCache(cacheName string) (CacheConfig, error) {
	for _, cache := range c.Caches {
		if cache.Name == cacheName {
			return cache, nil
		}
	}
	return CacheConfig{}, errors.New("cache not found in config")
}
