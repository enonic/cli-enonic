package util

import (
	"os"
	"encoding/json"
	"fmt"
)

type Config struct {
	Snapshot struct {
		Scheme string `json:"scheme"`
		Host   string `json:"host"`
		Port   uint16 `json:"port"`
		User   string `json:"user"`
		Pass   string `json:"pass"`
	} `json:"snapshot"`
}

func (c *Config) GetUrl() string {
	return fmt.Sprintf("%s://%s:%d", c.Snapshot.Scheme, c.Snapshot.Host, c.Snapshot.Port)
}

func (c *Config) GetAuth() string {
	return fmt.Sprintf("%s:%s", c.Snapshot.User, c.Snapshot.Pass)
}

var config,_ = readConfig()
func GetConfig() Config {
	return config
}

func readConfig() (Config, error) {
	var config Config
	configFile, err := os.Open("config.json")
	defer configFile.Close()
	if err != nil {
		return config, err
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config, nil
}
