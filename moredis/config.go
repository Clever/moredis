package moredis

import (
	"io/ioutil"
	"text/template"

	"gopkg.in/yaml.v2"
)

// Config is the config for the cache
type Config struct {
	Name        string             `yaml:"name"`
	Collections []CollectionConfig `yaml:"collections"`
}

// CollectionConfig is the config for a specific collection
type CollectionConfig struct {
	Collection string      `yaml:"collection"`
	Query      string      `yaml:"query"`
	Projection string      `yaml:"projection"`
	Sort       string      `yaml:"sort"`
	Maps       []MapConfig `yaml:"maps"`
}

// MapConfig is the config for a specific map.
type MapConfig struct {
	Name          string `yaml:"name"`
	Key           string `yaml:"key"`
	Value         string `yaml:"val"`
	HashKey       string
	KeyTemplate   *template.Template
	ValueTemplate *template.Template
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
