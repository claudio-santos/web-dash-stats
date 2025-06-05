package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Service definition for config and frontend
type Service struct {
	Name string `yaml:"name" json:"name"`
	URL  string `yaml:"url" json:"url"`
}

type Config struct {
	Services []Service `yaml:"services"`
}

// LoadConfig reads YAML config file into Config struct
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	return &cfg, err
}
