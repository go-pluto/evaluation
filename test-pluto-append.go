package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"crypto/x509"
	"io/ioutil"

	"github.com/emersion/go-imap/client"
	"github.com/numbleroot/pluto-evaluation/config"
	"github.com/numbleroot/pluto-evaluation/messages"
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
	err = imapClient.Login(config.Pluto.AppendTest.Name, config.Pluto.AppendTest.Password)
	if err != nil {
		log.Fatalf("[evaluation.TestPlutoAppend] Failed to login as '%s': %s\n", config.Pluto.AppendTest.Name, err.Error())
	}

	log.Printf("[evaluation.TestPlutoAppend] Logged in as '%s'.\n", config.Pluto.AppendTest.Name)

	// Log out on function exit.
	defer imapClient.Logout()

	// Take current time stamp and create log file name.
	logFileTime := time.Now()
	logFileName := fmt.Sprintf("results/pluto-append-test-%s.log", logFileTime.Format("2006-01-02-15-04-05"))

	// Attempt to create a test log file containing
	// measured test times.
	logFile, err := os.Create(logFileName)
	if err != nil {
		log.Fatalf("[evaluation.TestPlutoAppend] Failed to create test log file '%s': %s\n", logFileName, err.Error())
	}

	// Sync to storage and close on any exit.
	defer logFile.Close()
	defer logFile.Sync()

	// Prepare message to append.
	appendMsg := bytes.NewBufferString(messages.Msg01)

	for num := 0; num < 10; num++ {

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send mail message to server.
		err := imapClient.Append("INBOX", nil, time.Time{}, appendMsg)

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		// Now handle error if present.
		if err != nil {
			log.Fatalf("[evaluation.TestPlutoAppend] %d: Failed to send Msg01 to server: %s\n", num, err.Error())
		}

		log.Printf("Took %d - %d = %d nanoseconds.\n", timeEnd, timeStart, (timeEnd - timeStart))

		// Append log line to file.
		logFile.WriteString(fmt.Sprintf("%d, %d\n", num, (timeEnd - timeStart)))
	}

	// Calculate statistics and print them.
}
