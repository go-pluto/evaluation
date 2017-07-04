package utils

import (
	"fmt"

	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/go-pluto/evaluation/config"
	"github.com/go-pluto/pluto/crypto"
)

// Functions

// InitTLSConfigs returns pluto's and Dovecot's TLS
// config so that secure connections can be made from
// outside the systems.
func InitTLSConfigs(config *config.Config) (*tls.Config, *tls.Config, error) {

	// Create TLS config for pluto.
	plutoTLSConfig, err := crypto.NewPublicTLSConfig(config.Pluto.Distributor.CertLoc, config.Pluto.Distributor.KeyLoc)
	if err != nil {
		return nil, nil, err
	}

	// For tests, we currently need to build a custom
	// x509 cert pool to accept the self-signed public
	// distributor certificate.
	plutoTLSConfig.RootCAs = x509.NewCertPool()

	// Read distributor's public certificate in PEM format into memory.
	plutoRootCert, err := ioutil.ReadFile(config.Pluto.Distributor.CertLoc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load pluto's cert file: %s", err.Error())
	}

	// Append certificate to test client's root CA pool.
	ok := plutoTLSConfig.RootCAs.AppendCertsFromPEM(plutoRootCert)
	if ok != true {
		return nil, nil, fmt.Errorf("failed to append pluto's cert file")
	}

	// Create TLS config for Dovecot.
	dovecotTLSConfig := &tls.Config{
		// Unfortunately, we currently need to accept this.
		InsecureSkipVerify: true,
	}

	return plutoTLSConfig, dovecotTLSConfig, nil
}
