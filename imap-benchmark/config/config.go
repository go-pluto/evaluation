package config

import (
    "fmt"

    "github.com/BurntSushi/toml"
)

// Structs
// TODO: add proper comments

type Config struct {
	Server          Server
	Settings        Settings
	Session         Session
    NormDist        [5]float64
}

type Server struct {
	Hostname       string
	Port           string
	TLS            bool
}

type Settings struct {
	Threads        int
	Seed           int64
    Throttle       int
}

type Session struct {
	Distribution   [5]float64
    Length         int
}

// Functions

// LoadConfig takes in the path to the test config
func LoadConfig(configFile string) (*Config, error) {

	conf := new(Config)

	// Parse values from TOML file into struct.
	if _, err := toml.DecodeFile(configFile, conf); err != nil {
		return nil, fmt.Errorf("failed to read in TOML config file at '%s' with: %s\n", configFile, err.Error())
	}

    // calculate Distribution
    var sum float64 = 0

    for i := 0; i < 5; i++ {
        sum = sum + conf.Session.Distribution[i]
    }

    var normdist [5]float64

    for i := 0; i < 5; i++ {
        normdist[i] = conf.Session.Distribution[i] / sum
    }

    conf.NormDist = normdist

	return conf, nil
}
