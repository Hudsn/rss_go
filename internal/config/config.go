package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const filename = ".gatorconfig.json"

type Config struct {
	DBURL    string `json:"db_url"`
	UserName string `json:"current_user_name"`
	filePath string
}

func (c *Config) SetUser(userName string) error {
	c.UserName = userName
	b, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(c.filePath, b, 0644)
}

func ReadDefaultConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	home = filepath.Join(home, filename)
	if err != nil {
		return nil, err
	}
	return Read(home)
}

func Read(configPath string) (*Config, error) {

	var cfg *Config

	fileBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(fileBytes, &cfg)
	if err != nil {
		return nil, err
	}

	cfg.filePath = configPath

	return cfg, nil
}
