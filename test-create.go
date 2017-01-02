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

	log.Printf("[evaluation.TestCreate] Testing CREATE command on pluto and Dovecot...\n")

	// Make number of messages to send configurable.
	runsFlag := flag.Int("runs", 100, "Specify how many times the command of this test is to be sent to server.")
	flag.Parse()

	runs := *runsFlag

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
	plutoConn, err := tls.Dial("tcp", plutoIMAPAddr, plutoTLSConfig)
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Was unable to connect to remote pluto server: %s\n", err.Error())
	}

	// Create connection based on it.
	plutoC := conn.NewConn(plutoConn)

	// Consume mandatory IMAP greeting.
	_, err = plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Error during receiving initial server greeting: %s\n", err.Error())
	}

	// Log in as first user.
	err = plutoC.Send(fmt.Sprintf("createA LOGIN %s %s", config.Pluto.CreateTest.Name, config.Pluto.CreateTest.Password))
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Sending LOGIN to server failed with: %s\n", err.Error())
	}

	// Wait for success message.
	answer, err := plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Error during LOGIN as user %s: %s\n", config.Pluto.CreateTest.Name, err.Error())
	}

	if strings.HasPrefix(answer, "createA OK") != true {
		log.Fatalf("[evaluation.TestCreate] Server responded incorrectly to LOGIN: %s\n", answer)
	}

	log.Printf("[evaluation.TestCreate] Logged in as '%s'.\n", config.Pluto.CreateTest.Name)

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
	results := make([]int64, runs)

	log.Printf("[evaluation.TestCreate] Running tests on pluto...\n")

	for num := 1; num <= runs; num++ {

		i := num - 1

		// Prepare command to send.
		command := fmt.Sprintf("create%d CREATE test-mailbox-%d", num, num)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send CREATE commmand to server.
		err := plutoC.Send(command)
		if err != nil {
			log.Fatalf("[evaluation.TestCreate] %d: Failed during sending CREATE command: %s\n", num, err.Error())
		}

		// Receive answer to CREATE request.
		answer, err := plutoC.Receive()
		if err != nil {
			log.Fatalf("[evaluation.TestCreate] %d: Error receiving response to CREATE: %s\n", num, err.Error())
		}

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		if strings.Contains(answer, "CREATE completed") != true {
			log.Fatalf("[evaluation.TestCreate] %d: Server responded incorrectly to CREATE command: %s\n", num, answer)
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Store result in buffer.
		results[i] = rtt

		// Append log line to file.
		plutoLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	err = plutoC.Send("createZ LOGOUT")
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Error receiving first part of LOGOUT response: %s\n", err.Error())
	}

	// Receive next line from server.
	nextAnswer, err := plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Error receiving second part of LOGOUT response: %s\n", err.Error())
	}

	answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

	if strings.Contains(answer, "LOGOUT completed") != true {
		log.Fatalf("[evaluation.TestCreate] Server responded incorrectly to LOGOUT: %s\n", answer)
	}

	// Calculate statistics and print them.
	var sum int64 = 0
	for _, result := range results {
		sum += result
	}

	msAvg := (float64(sum) / float64(runs)) / float64(time.Millisecond)

	log.Printf("[evaluation.TestCreate] Done on pluto, sent %d create instructions, each took %f ms on average.\n\n", runs, msAvg)

	// Run tests on Dovecot.
	log.Printf("[evaluation.TestCreate] Connecting to Dovecot...\n")

	// Connect to remote Dovecot system.
	dovecotConn, err := net.Dial("tcp", dovecotIMAPAddr)
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Was unable to connect to remote Dovecot server: %s\n", err.Error())
	}

	// Create connection based on it.
	dovecotC := conn.NewConn(dovecotConn)

	// Consume mandatory IMAP greeting.
	_, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Error during receiving initial server greeting: %s\n", err.Error())
	}

	// Log in as first user.
	err = dovecotC.Send(fmt.Sprintf("createA LOGIN %s %s", config.Dovecot.CreateTest.Name, config.Dovecot.CreateTest.Password))
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Sending LOGIN to server failed with: %s\n", err.Error())
	}

	// Wait for success message.
	answer, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Error during LOGIN as user %s: %s\n", config.Dovecot.CreateTest.Name, err.Error())
	}

	if strings.HasPrefix(answer, "createA OK") != true {
		log.Fatalf("[evaluation.TestCreate] Server responded incorrectly to LOGIN: %s\n", answer)
	}

	log.Printf("[evaluation.TestCreate] Logged in as '%s'.\n", config.Dovecot.CreateTest.Name)

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

	for num := 1; num <= runs; num++ {

		i := num - 1

		// Prepare command to send.
		command := fmt.Sprintf("create%d CREATE test-mailbox-%d", num, num)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send CREATE commmand to server.
		err := dovecotC.Send(command)
		if err != nil {
			log.Fatalf("[evaluation.TestCreate] %d: Failed during sending CREATE command: %s\n", num, err.Error())
		}

		// Receive answer to CREATE request.
		answer, err := dovecotC.Receive()
		if err != nil {
			log.Fatalf("[evaluation.TestCreate] %d: Error receiving response to CREATE: %s\n", num, err.Error())
		}

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		if strings.Contains(answer, "Create completed") != true {
			log.Fatalf("[evaluation.TestCreate] %d: Server responded incorrectly to CREATE command: %s\n", num, answer)
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Store result in buffer.
		results[i] = rtt

		// Append log line to file.
		dovecotLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	err = dovecotC.Send("createZ LOGOUT")
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Error receiving first part of LOGOUT response: %s\n", err.Error())
	}

	// Receive next line from server.
	nextAnswer, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestCreate] Error receiving second part of LOGOUT response: %s\n", err.Error())
	}

	answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

	if strings.Contains(answer, "Logging out") != true {
		log.Fatalf("[evaluation.TestCreate] Server responded incorrectly to LOGOUT: %s\n", answer)
	}

	// Calculate statistics and print them.
	sum = 0
	for _, result := range results {
		sum += result
	}

	msAvg = 0
	msAvg = (float64(sum) / float64(runs)) / float64(time.Millisecond)

	log.Printf("[evaluation.TestCreate] Done on Dovecot, sent %d create instructions, each took %f ms on average.", runs, msAvg)
}
