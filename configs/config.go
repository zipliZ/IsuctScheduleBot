package configs

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Db        DbConfig  `yaml:"db"`
	BotToken  string    `yaml:"botToken"`
	Endpoints Endpoints `yaml:"endpoints"`
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

type Endpoints struct {
	Microservice string `yaml:"microservice"`
	Frontend     string `yaml:"frontend"`
}

func DecodeConfig(path string) Config {
	f, err := os.Open(path)
	if err != nil {
		log.Panic(err)
	}
	defer func(f *os.File) {
		closeErr := f.Close()
		if closeErr != nil {
			log.Panic(closeErr)
		}
	}(f)

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Panic(err)
	}
	return cfg
}
