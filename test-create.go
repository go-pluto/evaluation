package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"crypto/x509"
	"io/ioutil"

	"github.com/emersion/go-imap/client"
	"github.com/numbleroot/pluto-evaluation/config"
	"github.com/numbleroot/pluto/crypto"
)

// Functions

func main() {

	var err error
	var plutoClient *client.Client
	var dovecotClient *client.Client

	log.Printf("[evaluation.TestCreate] Testing CREATE command on pluto and Dovecot...\n")

	// Read configuration from file.
	config, err := config.LoadConfig("test-config-aws.toml")
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Error loading config: %s\n", err.Error())
	}

	// Create TLS config.
	plutoTLSConfig, err := crypto.NewPublicTLSConfig(config.Pluto.Distributor.CertLoc, config.Pluto.Distributor.KeyLoc)
	if err != nil {
		log.Fatal(err)
	}

	// For tests, we currently need to build a custom
	// x509 cert pool to accept the self-signed public
	// distributor certificate.
	plutoTLSConfig.RootCAs = x509.NewCertPool()

	// Read distributor's public certificate in PEM format into memory.
	plutoRootCert, err := ioutil.ReadFile(config.Pluto.Distributor.CertLoc)
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Failed to load cert file: %s\n", err.Error())
	}

	// Append certificate to test client's root CA pool.
	if ok := plutoTLSConfig.RootCAs.AppendCertsFromPEM(plutoRootCert); !ok {
		log.Fatalf("[evaluation.TestCreate] Failed to append cert.\n")
	}

	// Create connection string to connect to pluto and Dovecot.
	plutoIMAPAddr := fmt.Sprintf("%s:%s", config.Pluto.IP, config.Pluto.Port)
	dovecotIMAPAddr := fmt.Sprintf("%s:%s", config.Dovecot.IP, config.Dovecot.Port)

	log.Printf("[evaluation.TestCreate] Connecting to pluto...\n")

	// Connect to remote pluto system.
	if config.Pluto.TLS {
		plutoClient, err = client.DialTLS(plutoIMAPAddr, plutoTLSConfig)
	} else {
		plutoClient, err = client.Dial(plutoIMAPAddr)
	}

	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Was unable to connect to remote pluto server: %s\n", err.Error())
	}

	// Log in as first user.
	err = plutoClient.Login(config.Pluto.AppendTest.Name, config.Pluto.AppendTest.Password)
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Failed to login as '%s': %s\n", config.Pluto.AppendTest.Name, err.Error())
	}

	log.Printf("[evaluation.TestCreate] Logged in as '%s'.\n", config.Pluto.AppendTest.Name)

	// Take current time stamp and create log file name.
	logFileTime := time.Now()
	plutoLogFileName := fmt.Sprintf("results/pluto-create-%s.log", logFileTime.Format("2006-01-02-15-04-05"))

	// Attempt to create a test log file containing
	// measured test times for pluto system.
	plutoLogFile, err := os.Create(plutoLogFileName)
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Failed to create test log file '%s': %s\n", plutoLogFileName, err.Error())
	}

	// Sync to storage and close on any exit.
	defer plutoLogFile.Close()
	defer plutoLogFile.Sync()

	// Prepend file with meta information about this test.
	plutoLogFile.WriteString(fmt.Sprintf("Subject: CREATE\nPlatform: pluto\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

	// Prepare buffer to append individual results to.
	runs := 100
	results := make([]int64, runs)

	log.Printf("[evaluation.TestCreate] Running tests on pluto...\n")

	for num := 0; num < runs; num++ {

		// Mailbox name to create.
		mailboxName := fmt.Sprintf("test-mailbox-%d", num)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Create test mailbox on remote server.
		err := plutoClient.Create(mailboxName)

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		// Now handle error if present.
		if err != nil {
			log.Fatalf("[evaluation.TestCreate] %d: Failed to create test mailbox: %s\n", num, err.Error())
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Store result in buffer.
		results[num] = rtt

		// Append log line to file.
		plutoLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	plutoClient.Logout()

	// Calculate statistics and print them.
	var sum int64 = 0
	for _, result := range results {
		sum += result
	}

	msAvg := (float64(sum) / float64(runs)) / float64(time.Millisecond)

	log.Printf("[evaluation.TestCreate] Done on pluto, created %d mailboxes, each took %f ms on average.\n\n", runs, msAvg)

	// Run tests on Dovecot.
	log.Printf("[evaluation.TestCreate] Connecting to Dovecot...\n")

	// Connect to remote Dovecot system.
	if config.Dovecot.TLS {
		dovecotClient, err = client.DialTLS(dovecotIMAPAddr, nil)
	} else {
		dovecotClient, err = client.Dial(dovecotIMAPAddr)
	}

	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Was unable to connect to remote Dovecot server: %s\n", err.Error())
	}

	// Log in as first user.
	err = dovecotClient.Login(config.Dovecot.AppendTest.Name, config.Dovecot.AppendTest.Password)
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Failed to login as '%s': %s\n", config.Dovecot.AppendTest.Name, err.Error())
	}

	log.Printf("[evaluation.TestCreate] Logged in as '%s'.\n", config.Dovecot.AppendTest.Name)

	// Prepare log file name for Dovecot.
	dovecotLogFileName := fmt.Sprintf("results/dovecot-create-%s.log", logFileTime.Format("2006-01-02-15-04-05"))

	// Attempt to create a test log file containing
	// measured test times for Dovecot system.
	dovecotLogFile, err := os.Create(dovecotLogFileName)
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Failed to create test log file '%s': %s\n", dovecotLogFileName, err.Error())
	}

	// Sync to storage and close on any exit.
	defer dovecotLogFile.Close()
	defer dovecotLogFile.Sync()

	// Prepend file with meta information about this test.
	dovecotLogFile.WriteString(fmt.Sprintf("Subject: CREATE\nPlatform: Dovecot\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

	log.Printf("[evaluation.TestCreate] Running tests on Dovecot...\n")

	// Reset results slice.
	results = make([]int64, runs)

	for num := 0; num < runs; num++ {

		// Mailbox name to create.
		mailboxName := fmt.Sprintf("test-mailbox-%d", num)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Create test mailbox on remote server.
		err := dovecotClient.Create(mailboxName)

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		// Now handle error if present.
		if err != nil {
			log.Fatalf("[evaluation.TestCreate] %d: Failed to create test mailbox: %s\n", num, err.Error())
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Store result in buffer.
		results[num] = rtt

		// Append log line to file.
		dovecotLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	dovecotClient.Logout()

	// Calculate statistics and print them.
	sum = 0
	for _, result := range results {
		sum += result
	}

	msAvg = 0
	msAvg = (float64(sum) / float64(runs)) / float64(time.Millisecond)

	log.Printf("[evaluation.TestCreate] Done on Dovecot, created %d mailboxes, each took %f ms on average.\n\n", runs, msAvg)
}
