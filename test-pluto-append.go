package main

import (
	"fmt"
	"log"

	"crypto/x509"
	"io/ioutil"

	"github.com/emersion/go-imap/client"
	"github.com/numbleroot/pluto-evaluation/config"
	"github.com/numbleroot/pluto/crypto"
)

// Functions

func main() {

	var err error
	var imapClient *client.Client

	// Read configuration from file.
	config, err := config.LoadConfig("test-config-aws.toml")
	if err != nil {
		log.Fatalf("[evaluation.TestPlutoAppend] Error loading config: %s\n", err.Error())
	}

	// Create TLS config.
	tlsConfig, err := crypto.NewPublicTLSConfig(config.Pluto.Distributor.CertLoc, config.Pluto.Distributor.KeyLoc)
	if err != nil {
		log.Fatal(err)
	}

	// For tests, we currently need to build a custom
	// x509 cert pool to accept the self-signed public
	// distributor certificate.
	tlsConfig.RootCAs = x509.NewCertPool()

	// Read distributor's public certificate in PEM format into memory.
	rootCert, err := ioutil.ReadFile(config.Pluto.Distributor.CertLoc)
	if err != nil {
		log.Fatalf("[evaluation.TestPlutoAppend] Failed to load cert file: %s\n", err.Error())
	}

	// Append certificate to test client's root CA pool.
	if ok := tlsConfig.RootCAs.AppendCertsFromPEM(rootCert); !ok {
		log.Fatalf("[evaluation.TestPlutoAppend] Failed to append cert.\n")
	}

	// Create connection string to connect to.
	imapAddr := fmt.Sprintf("%s:%s", config.Pluto.IP, config.Pluto.Port)

	// Connect to remote pluto system.
	if config.Pluto.TLS {
		imapClient, err = client.DialTLS(imapAddr, tlsConfig)
	} else {
		imapClient, err = client.Dial(imapAddr)
	}

	if err != nil {
		log.Fatalf("[evaluation.TestPlutoAppend] Was unable to connect to remote IMAP server: %s\n", err.Error())
	}

	// Log in as first user.
	err = imapClient.Login("user0", "password0")
	if err != nil {
		log.Fatalf("[evaluation.TestPlutoAppend] Failed to login as 'user0': %s\n", err.Error())
	}

	// Log out on function exit.
	defer imapClient.Logout()

	// Select INBOX as mailbox.
	inbox, err := imapClient.Select("INBOX", false)
	if err != nil {
		log.Fatalf("[evaluation.TestPlutoAppend] Error during selecting 'INBOX': %s\n", err.Error())
	}

	log.Printf("inbox: %#v\n", inbox)

	// For each mail to append:
	// * take current time stamp A
	// * prepare log line
	// * send mail to remote system
	// * wait for response
	// * log reponse time stamp B
	// * calculate rtt = B - A
	// * finish log line and append to test log

	// Calculate statistics and print them.

	// Close log file and exit.
}
