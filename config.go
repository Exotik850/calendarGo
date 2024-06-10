package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	StartAddress string  `json:"start_address"`
	EndAddress   *string `json:"end_address,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	// Load the configuration from the file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil

}
