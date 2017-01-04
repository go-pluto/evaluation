package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/numbleroot/pluto-evaluation/config"
	"github.com/numbleroot/pluto-evaluation/conn"
	"github.com/numbleroot/pluto/crypto"
)

// Functions

func main() {

	log.Printf("[evaluation.TestStore] Testing STORE command on pluto and Dovecot...\n")

	// Make test config file location and number of messages
	// to send per test configurable.
	configFlag := flag.String("config", "test-config.toml", "Specify location of config file that describes test setup configuration.")
	runsFlag := flag.Int("runs", 100, "Specify how many times the command of this test is to be sent to server.")
	flag.Parse()

	runs := *runsFlag

	// Read configuration from file.
	config, err := config.LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error loading config: %s\n", err.Error())
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
		log.Fatalf("[evaluation.TestStore] Failed to load cert file: %s\n", err.Error())
	}

	// Append certificate to test client's root CA pool.
	if ok := plutoTLSConfig.RootCAs.AppendCertsFromPEM(plutoRootCert); !ok {
		log.Fatalf("[evaluation.TestStore] Failed to append cert.\n")
	}

	// Create connection string to connect to pluto and Dovecot.
	plutoIMAPAddr := fmt.Sprintf("%s:%s", config.Pluto.IP, config.Pluto.Port)
	dovecotIMAPAddr := fmt.Sprintf("%s:%s", config.Dovecot.IP, config.Dovecot.Port)

	log.Printf("[evaluation.TestStore] Connecting to pluto...\n")

	// Connect to remote pluto system.
	plutoConn, err := tls.Dial("tcp", plutoIMAPAddr, plutoTLSConfig)
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Was unable to connect to remote pluto server: %s\n", err.Error())
	}

	// Create connection based on it.
	plutoC := conn.NewConn(plutoConn)

	// Consume mandatory IMAP greeting.
	_, err = plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error during receiving initial server greeting: %s\n", err.Error())
	}

	// Log in as first user.
	err = plutoC.Send(fmt.Sprintf("storeA LOGIN %s %s", config.Pluto.StoreTest.Name, config.Pluto.StoreTest.Password))
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Sending LOGIN to server failed with: %s\n", err.Error())
	}

	// Wait for success message.
	answer, err := plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error during LOGIN as user %s: %s\n", config.Pluto.StoreTest.Name, err.Error())
	}

	if strings.HasPrefix(answer, "storeA OK") != true {
		log.Fatalf("[evaluation.TestStore] Server responded incorrectly to LOGIN: %s\n", answer)
	}

	log.Printf("[evaluation.TestStore] Logged in as '%s'.\n", config.Pluto.StoreTest.Name)

	// Select INBOX for all following commands.
	err = plutoC.Send("storeB SELECT INBOX")
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Sending SELECT to server failed with: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error receiving first part of SELECT response: %s\n", err.Error())
	}

	// As long as the IMAP command termination indicator
	// was not yet received, continue to append answers.
	for (strings.Contains(answer, "completed") != true) &&
		(strings.Contains(answer, "BAD") != true) &&
		(strings.Contains(answer, "NO") != true) {

		// Receive next line from distributor.
		nextAnswer, err := plutoC.Receive()
		if err != nil {
			log.Fatalf("[evaluation.TestStore] Error receiving next part of SELECT response: %s\n", err.Error())
		}

		answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)
	}

	if strings.Contains(answer, "storeB OK") != true {
		log.Fatalf("[evaluation.TestStore] Server responded incorrectly to SELECT: %s\n", answer)
	}

	log.Printf("[evaluation.TestStore] Selected INBOX for further commands.\n")

	// Take current time stamp and create log file name.
	logFileTime := time.Now()
	plutoLogFileName := fmt.Sprintf("results/pluto-store-%s.log", logFileTime.Format("2006-01-02-15-04-05"))

	// Attempt to create a test log file containing
	// measured test times for pluto system.
	plutoLogFile, err := os.Create(plutoLogFileName)
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Failed to create test log file '%s': %s\n", plutoLogFileName, err.Error())
	}

	// Sync to storage and close on any exit.
	defer plutoLogFile.Close()
	defer plutoLogFile.Sync()

	// Prepend file with meta information about this test.
	plutoLogFile.WriteString(fmt.Sprintf("Subject: STORE\nPlatform: pluto\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

	// Prepare buffer to append individual results to.
	results := make([]int64, runs)

	log.Printf("[evaluation.TestStore] Running tests on pluto...\n")

	for num := 1; num <= runs; num++ {

		i := num - 1

		// Prepare command to send.
		command := fmt.Sprintf("store%d STORE %d +FLAGS.SILENT (\\Seen \\Deleted)", num, num)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send STORE commmand to server.
		err := plutoC.Send(command)
		if err != nil {
			log.Fatalf("[evaluation.TestStore] %d: Failed during sending STORE command: %s\n", num, err.Error())
		}

		// Receive answer to STORE request.
		answer, err := plutoC.Receive()
		if err != nil {
			log.Fatalf("[evaluation.TestStore] %d: Error receiving response to STORE: %s\n", num, err.Error())
		}

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		if strings.Contains(answer, "STORE completed") != true {
			log.Fatalf("[evaluation.TestStore] %d: Server responded incorrectly to STORE command: %s\n", num, answer)
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Store result in buffer.
		results[i] = rtt

		// Append log line to file.
		plutoLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	err = plutoC.Send("storeZ LOGOUT")
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error receiving first part of LOGOUT response: %s\n", err.Error())
	}

	// Receive next line from server.
	nextAnswer, err := plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error receiving second part of LOGOUT response: %s\n", err.Error())
	}

	answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

	if strings.Contains(answer, "LOGOUT completed") != true {
		log.Fatalf("[evaluation.TestStore] Server responded incorrectly to LOGOUT: %s\n", answer)
	}

	// Calculate statistics and print them.
	var sum int64 = 0
	for _, result := range results {
		sum += result
	}

	msAvg := (float64(sum) / float64(runs)) / float64(time.Millisecond)

	log.Printf("[evaluation.TestStore] Done on pluto, sent %d store instructions, each took %f ms on average.\n\n", runs, msAvg)

	// Run tests on Dovecot.
	log.Printf("[evaluation.TestStore] Connecting to Dovecot...\n")

	// Connect to remote Dovecot system.
	dovecotConn, err := net.Dial("tcp", dovecotIMAPAddr)
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Was unable to connect to remote Dovecot server: %s\n", err.Error())
	}

	// Create connection based on it.
	dovecotC := conn.NewConn(dovecotConn)

	// Consume mandatory IMAP greeting.
	_, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error during receiving initial server greeting: %s\n", err.Error())
	}

	// Log in as first user.
	err = dovecotC.Send(fmt.Sprintf("storeA LOGIN %s %s", config.Dovecot.StoreTest.Name, config.Dovecot.StoreTest.Password))
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Sending LOGIN to server failed with: %s\n", err.Error())
	}

	// Wait for success message.
	answer, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error during LOGIN as user %s: %s\n", config.Dovecot.StoreTest.Name, err.Error())
	}

	if strings.HasPrefix(answer, "storeA OK") != true {
		log.Fatalf("[evaluation.TestStore] Server responded incorrectly to LOGIN: %s\n", answer)
	}

	log.Printf("[evaluation.TestStore] Logged in as '%s'.\n", config.Dovecot.StoreTest.Name)

	// Select INBOX for all following commands.
	err = dovecotC.Send("storeB SELECT INBOX")
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Sending SELECT to server failed with: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error receiving first part of SELECT response: %s\n", err.Error())
	}

	// As long as the IMAP command termination indicator
	// was not yet received, continue to append answers.
	for (strings.Contains(answer, "completed") != true) &&
		(strings.Contains(answer, "BAD") != true) &&
		(strings.Contains(answer, "NO") != true) {

		// Receive next line from distributor.
		nextAnswer, err := dovecotC.Receive()
		if err != nil {
			log.Fatalf("[evaluation.TestStore] Error receiving next part of SELECT response: %s\n", err.Error())
		}

		answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)
	}

	if strings.Contains(answer, "storeB OK") != true {
		log.Fatalf("[evaluation.TestStore] Server responded incorrectly to SELECT: %s\n", answer)
	}

	log.Printf("[evaluation.TestStore] Selected INBOX for further commands.\n")

	// Prepare log file name for Dovecot.
	dovecotLogFileName := fmt.Sprintf("results/dovecot-store-%s.log", logFileTime.Format("2006-01-02-15-04-05"))

	// Attempt to create a test log file containing
	// measured test times for Dovecot system.
	dovecotLogFile, err := os.Create(dovecotLogFileName)
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Failed to create test log file '%s': %s\n", dovecotLogFileName, err.Error())
	}

	// Sync to storage and close on any exit.
	defer dovecotLogFile.Close()
	defer dovecotLogFile.Sync()

	// Prepend file with meta information about this test.
	dovecotLogFile.WriteString(fmt.Sprintf("Subject: STORE\nPlatform: Dovecot\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

	log.Printf("[evaluation.TestStore] Running tests on Dovecot...\n")

	// Reset results slice.
	results = make([]int64, runs)

	for num := 1; num <= runs; num++ {

		i := num - 1

		// Prepare command to send.
		command := fmt.Sprintf("store%d STORE %d +FLAGS.SILENT (\\Seen \\Deleted)", num, num)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send STORE commmand to server.
		err := dovecotC.Send(command)
		if err != nil {
			log.Fatalf("[evaluation.TestStore] %d: Failed during sending STORE command: %s\n", num, err.Error())
		}

		// Receive answer to STORE request.
		answer, err := dovecotC.Receive()
		if err != nil {
			log.Fatalf("[evaluation.TestStore] %d: Error receiving response to STORE: %s\n", num, err.Error())
		}

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		if strings.Contains(answer, "Store completed") != true {
			log.Fatalf("[evaluation.TestStore] %d: Server responded incorrectly to STORE command: %s\n", num, answer)
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Store result in buffer.
		results[i] = rtt

		// Append log line to file.
		dovecotLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	err = dovecotC.Send("storeZ LOGOUT")
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error receiving first part of LOGOUT response: %s\n", err.Error())
	}

	// Receive next line from server.
	nextAnswer, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestStore] Error receiving second part of LOGOUT response: %s\n", err.Error())
	}

	answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

	if strings.Contains(answer, "Logging out") != true {
		log.Fatalf("[evaluation.TestStore] Server responded incorrectly to LOGOUT: %s\n", answer)
	}

	// Calculate statistics and print them.
	sum = 0
	for _, result := range results {
		sum += result
	}

	msAvg = 0
	msAvg = (float64(sum) / float64(runs)) / float64(time.Millisecond)

	log.Printf("[evaluation.TestStore] Done on Dovecot, sent %d store instructions, each took %f ms on average.", runs, msAvg)
}
