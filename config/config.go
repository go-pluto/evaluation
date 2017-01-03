package config

import (
	"fmt"

	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Structs

// Config holds all information parsed from
// supplied config file.
type Config struct {
	Pluto   Pluto
	Dovecot Dovecot
}

// Pluto defines the relevant information in
// order to connect to a pluto system to test.
type Pluto struct {
	IP             string
	Port           string
	TLS            bool
	RootCertLoc    string
	Distributor    Distributor
	AppendTest     User
	CreateTest     User
	DeleteTest     User
	StoreTest      User
	ConcurrentTest ConcurrentTest
}

// Dovecot contains IP and port of the comparison
// Dovecot system to test.
type Dovecot struct {
	IP             string
	Port           string
	TLS            bool
	AppendTest     User
	CreateTest     User
	DeleteTest     User
	StoreTest      User
	ConcurrentTest ConcurrentTest
}

// Distributor holds paths to self-signed certificates
// to connect securely to pluto's distributor.
type Distributor struct {
	CertLoc string
	KeyLoc  string
}

// User carries authentication information for a test
// user in system to be tested.
type User struct {
	Name     string
	Password string
}

// ConcurrentTest contains a slice of users to test
// concurrent access for.
type ConcurrentTest struct {
	User []User
}

// Functions

// LoadConfig takes in the path to the test config
// file of pluto and Dovecot system in TOML syntax
// and fills above structs.
func LoadConfig(configFile string) (*Config, error) {

	conf := new(Config)

	// Parse values from TOML file into struct.
	if _, err := toml.DecodeFile(configFile, conf); err != nil {
		return nil, fmt.Errorf("failed to read in TOML config file at '%s' with: %s\n", configFile, err.Error())
	}

	// Retrieve absolute path of pluto-evaluation directory.
	absEvalPath, err := filepath.Abs("./")
	if err != nil {
		return nil, fmt.Errorf("could not get absolute path of current directory: %s\n", err.Error())
	}

	// Prefix each relative path in config with just
	// obtained absolute path to pluto-evaluation directory.

	// Pluto.RootCertLoc
	if filepath.IsAbs(conf.Pluto.RootCertLoc) != true {
		conf.Pluto.RootCertLoc = filepath.Join(absEvalPath, conf.Pluto.RootCertLoc)
	}

	// Pluto.Distributor.CertLoc
	if filepath.IsAbs(conf.Pluto.Distributor.CertLoc) != true {
		conf.Pluto.Distributor.CertLoc = filepath.Join(absEvalPath, conf.Pluto.Distributor.CertLoc)
	}

	// Pluto.Distributor.KeyLoc
	if filepath.IsAbs(conf.Pluto.Distributor.KeyLoc) != true {
		conf.Pluto.Distributor.KeyLoc = filepath.Join(absEvalPath, conf.Pluto.Distributor.KeyLoc)
	}

	return conf, nil
}
