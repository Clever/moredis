package main

import (
	"errors"
	"io/ioutil"

	"gopkg.in/v1/yaml"
)

type config struct {
	Caches []cacheConfig `yaml:"caches"`
}

type cacheConfig struct {
	Name        string             `yaml:"name"`
	Collections []collectionConfig `yaml:"collections"`
}

type collectionConfig struct {
	Collection string      `yaml:"collection"`
	Query      string      `yaml:"query"`
	Maps       []mapConfig `yaml:"maps"`
}

type mapConfig struct {
	Name    string `yaml:"name"`
	Key     string `yaml:"key"`
	Value   string `yaml:"val"`
	HashKey string
}

// LoadConfig takes a path to a config yaml file and loads it into the appropriate structs.
func LoadConfig(path string) (config, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return config{}, err
	}

	var conf config
	if err = yaml.Unmarshal(raw, &conf); err != nil {
		return config{}, err
	}

	return conf, nil
}

// GetCache returns the config for a specific cache inside the moredis config.
func (c config) GetCache(cacheName string) (cacheConfig, error) {
	for _, cache := range c.Caches {
		if cache.Name == cacheName {
			return cache, nil
		}
	}
	return cacheConfig{}, errors.New("Cache not found in config")
}
