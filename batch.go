package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type batchSpec struct {
	Name          string    `yaml:"name"`
	IsDistributed bool      `yaml:"distributed"`
	IsWorker      bool      `yaml:"worker"`
	Url           string    `yaml:"url"`
	Runs          []runConf `yaml:"runs"`
	Output        string    `yaml:"output"`
	Format        string    `yaml:"format"`
}

type runConf struct {
	Requests    int    `yaml:"requests"`
	Concurrency int    `yaml:"concurrency"`
	CustomURL   string `yaml:"url"`
	CustomID    string `yaml:"id"`
}

func (conf *batchSpec) getConf(filename string) *batchSpec {

	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("Error reading file: %v\n", err)
	}
	err = yaml.Unmarshal(fileBytes, conf)
	if err != nil {
		log.Fatalf("Error unmarshalling yaml: %v\n", err)
	}

	return conf
}
