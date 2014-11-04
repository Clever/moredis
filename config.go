package main

import (
	"errors"
	"io/ioutil"

	"gopkg.in/v1/yaml"
)

type Config struct {
	Caches []CacheConfig `yaml:"caches"`
}

type CacheConfig struct {
	Name        string             `yaml:"name"`
	Collections []CollectionConfig `yaml:"collections"`
}

type CollectionConfig struct {
	Collection string      `yaml:"collection"`
	Query      string      `yaml:"query"`
	Maps       []MapConfig `yaml:"maps"`
}

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

	var config Config
	if err = yaml.Unmarshal(raw, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}

// GetCache returns the config for a specific cache inside the moredis config.
func (c Config) GetCache(cache_name string) (CacheConfig, error) {
	for _, cache := range c.Caches {
		if cache.Name == cache_name {
			return cache, nil
		}
	}
	return CacheConfig{}, errors.New("Cache not found in config")
}
