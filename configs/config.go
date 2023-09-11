package configs

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Db  DbConfig  `yaml:"db"`
	Bot BotConfig `yaml:"bot"`
}

type DbConfig struct {
	User   string `yaml:"user"`
	Pass   string `yaml:"pass"`
	Host   string `yaml:"host"`
	Port   string `yaml:"port"`
	DbName string `yaml:"dbName"`
}

type BotConfig struct {
	Token string `yaml:"token"`
}

func DecodeConfig() Config {
	f, err := os.Open("./config.yaml")
	if err != nil {
		panic(err)
	}
	defer func(f *os.File) {
		closeErr := f.Close()
		if closeErr != nil {
			panic(closeErr)
		}
	}(f)

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		panic(err)
	}
	return cfg
}
