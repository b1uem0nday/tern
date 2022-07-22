package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

const DefConfigPath = "config.yaml"

type (
	Db struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Scheme   string `yaml:"scheme"`
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
	}

	Config struct {
		Path string `yaml:"path"`
		Base Db     `yaml:"base"`
	}
)

var defaultConfig = Config{
	Path: "./migrations",
	Base: Db{
		Username: "user",
		Password: "P@ssw0rd",
		Scheme:   "postgres",
		Host:     "localhost",
		Port:     "5432",
	},
}

func LoadConfig(path string) (*Config, error) {
	if _, err := os.Stat(path); err != nil {
		return &defaultConfig, err
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return &defaultConfig, err
	}
	var config Config
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return &defaultConfig, err
	}
	return &config, nil

}

func (c *Config) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", c.Base.Username, c.Base.Password, c.Base.Host, c.Base.Port, c.Base.Scheme)
}
