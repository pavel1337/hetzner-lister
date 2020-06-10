package main

import (
	"flag"
	"io/ioutil"

	"github.com/ghodss/yaml"
)

type Config struct {
	CloudTokens []string `json:"cloud_tokens"`
	RobotCreds  []struct {
		User     string `json:"user"`
		Password string `json:"password"`
	} `json:"robot_creds"`
}

func parseConfig(p string) (*Config, error) {
	var c Config
	rawConfig, err := ioutil.ReadFile(p)
	if err != nil {
		flag.Usage()
		return nil, err
	}
	err = yaml.Unmarshal(rawConfig, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
