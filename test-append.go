package main

import (
	"bytes"
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
	"github.com/numbleroot/pluto-evaluation/messages"
	"github.com/numbleroot/pluto/crypto"
)

// Functions

func main() {

	log.Printf("[evaluation.TestAppend] Testing APPEND command on pluto and Dovecot...\n")

	// Make number of messages to send configurable.
	runsFlag := flag.Int("runs", 100, "Specify how many times the command of this test is to be sent to server.")
	flag.Parse()

	runs := *runsFlag

	// Read configuration from file.
	config, err := config.LoadConfig("test-config-aws.toml")
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Error loading config: %s\n", err.Error())
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
		log.Fatalf("[evaluation.TestAppend] Failed to load cert file: %s\n", err.Error())
	}

	// Append certificate to test client's root CA pool.
	if ok := plutoTLSConfig.RootCAs.AppendCertsFromPEM(plutoRootCert); !ok {
		log.Fatalf("[evaluation.TestAppend] Failed to append cert.\n")
	}

	// Create connection string to connect to pluto and Dovecot.
	plutoIMAPAddr := fmt.Sprintf("%s:%s", config.Pluto.IP, config.Pluto.Port)
	dovecotIMAPAddr := fmt.Sprintf("%s:%s", config.Dovecot.IP, config.Dovecot.Port)

	log.Printf("[evaluation.TestAppend] Connecting to pluto...\n")

	// Connect to remote pluto system.
	plutoConn, err := tls.Dial("tcp", plutoIMAPAddr, plutoTLSConfig)
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Was unable to connect to remote pluto server: %s\n", err.Error())
	}

	// Create connection based on it.
	plutoC := conn.NewConn(plutoConn)

	// Consume mandatory IMAP greeting.
	_, err = plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Error during receiving initial server greeting: %s\n", err.Error())
	}

	// Log in as first user.
	err = plutoC.Send(fmt.Sprintf("a LOGIN %s %s", config.Pluto.AppendTest.Name, config.Pluto.AppendTest.Password))
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Sending LOGIN to server failed with: %s\n", err.Error())
	}

	// Wait for success message.
	answer, err := plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Error during LOGIN as user %s: %s\n", config.Pluto.AppendTest.Name, err.Error())
	}

	if strings.HasPrefix(answer, "a OK ") != true {
		log.Fatalf("[evaluation.TestAppend] Server responded incorrectly to LOGIN: %s\n", answer)
	}

	log.Printf("[evaluation.TestAppend] Logged in as '%s'.\n", config.Pluto.AppendTest.Name)

	// Take current time stamp and create log file name.
	logFileTime := time.Now()
	plutoLogFileName := fmt.Sprintf("results/pluto-append-%s.log", logFileTime.Format("2006-01-02-15-04-05"))

	// Attempt to create a test log file containing
	// measured test times for pluto system.
	plutoLogFile, err := os.Create(plutoLogFileName)
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Failed to create test log file '%s': %s\n", plutoLogFileName, err.Error())
	}

	// Sync to storage and close on any exit.
	defer plutoLogFile.Close()
	defer plutoLogFile.Sync()

	// Prepend file with meta information about this test.
	plutoLogFile.WriteString(fmt.Sprintf("Subject: APPEND\nPlatform: pluto\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

	// Prepare buffer to append individual results to.
	results := make([]int64, runs)

	// Prepare message to append.
	appendMsg := bytes.NewBufferString(messages.Msg01)
	appendMsgSize := appendMsg.Len()

	log.Printf("[evaluation.TestAppend] Running tests on pluto...\n")

	for num := 0; num < runs; num++ {

		// Prepare command to send.
		command := fmt.Sprintf("%d APPEND INBOX {%d}", num, appendMsgSize)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send APPEND commmand to server.
		err := plutoC.Send(command)
		if err != nil {
			log.Fatalf("[evaluation.TestAppend] %d: Failed during sending APPEND command: %s\n", num, err.Error())
		}

		// Receive answer to APPEND request.
		answer, err := plutoC.Receive()
		if err != nil {
			log.Fatalf("[evaluation.TestAppend] %d: Error receiving response to APPEND: %s\n", num, err.Error())
		}

		if answer != "+ Ready for literal data" {
			log.Fatalf("[evaluation.TestAppend] %d: Did not receive continuation command from server: %s\n", num, answer)
		}

		// Send mail message without additional newline.
		_, err = fmt.Fprintf(plutoC.Conn, "%s", appendMsg)
		if err != nil {
			log.Fatalf("[evaluation.TestAppend] %d: Sending mail message to server failed with: %s\n", num, err.Error())
		}

		// Receive answer to message transfer.
		answer, err = plutoC.Receive()
		if err != nil {
			log.Fatalf("[evaluation.TestAppend] %d: Error during receiving response to APPEND: %s\n", num, err.Error())
		}

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		if strings.Contains(answer, "APPEND completed") != true {
			log.Fatalf("[evaluation.TestAppend] %d: Server responded incorrectly to APPEND command: %s\n", num, answer)
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Store result in buffer.
		results[num] = rtt

		// Append log line to file.
		plutoLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	err = plutoC.Send("z LOGOUT")
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Error receiving first part of LOGOUT response: %s\n", err.Error())
	}

	// Receive next line from server.
	nextAnswer, err := plutoC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Error receiving second part of LOGOUT response: %s\n", err.Error())
	}

	answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

	if strings.Contains(answer, "LOGOUT completed") != true {
		log.Fatalf("[evaluation.TestAppend] Server responded incorrectly to LOGOUT: %s\n", answer)
	}

	// Calculate statistics and print them.
	var sum int64 = 0
	for _, result := range results {
		sum += result
	}

	msAvg := (float64(sum) / float64(runs)) / float64(time.Millisecond)

	log.Printf("[evaluation.TestAppend] Done on pluto, sent %d messages, each took %f ms on average.\n\n", runs, msAvg)

	// Run tests on Dovecot.
	log.Printf("[evaluation.TestAppend] Connecting to Dovecot...\n")

	// Connect to remote Dovecot system.
	dovecotConn, err := net.Dial("tcp", dovecotIMAPAddr)
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Was unable to connect to remote Dovecot server: %s\n", err.Error())
	}

	// Create connection based on it.
	dovecotC := conn.NewConn(dovecotConn)

	// Consume mandatory IMAP greeting.
	_, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Error during receiving initial server greeting: %s\n", err.Error())
	}

	// Log in as first user.
	err = dovecotC.Send(fmt.Sprintf("a LOGIN %s %s", config.Dovecot.AppendTest.Name, config.Dovecot.AppendTest.Password))
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Sending LOGIN to server failed with: %s\n", err.Error())
	}

	// Wait for success message.
	answer, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Error during LOGIN as user %s: %s\n", config.Dovecot.AppendTest.Name, err.Error())
	}

	if strings.HasPrefix(answer, "a OK ") != true {
		log.Fatalf("[evaluation.TestAppend] Server responded incorrectly to LOGIN: %s\n", answer)
	}

	log.Printf("[evaluation.TestAppend] Logged in as '%s'.\n", config.Dovecot.AppendTest.Name)

	// Prepare log file name for Dovecot.
	dovecotLogFileName := fmt.Sprintf("results/dovecot-append-%s.log", logFileTime.Format("2006-01-02-15-04-05"))

	// Attempt to create a test log file containing
	// measured test times for Dovecot system.
	dovecotLogFile, err := os.Create(dovecotLogFileName)
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Failed to create test log file '%s': %s\n", dovecotLogFileName, err.Error())
	}

	// Sync to storage and close on any exit.
	defer dovecotLogFile.Close()
	defer dovecotLogFile.Sync()

	// Prepend file with meta information about this test.
	dovecotLogFile.WriteString(fmt.Sprintf("Subject: APPEND\nPlatform: Dovecot\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

	log.Printf("[evaluation.TestAppend] Running tests on Dovecot...\n")

	// Reset results slice.
	results = make([]int64, runs)

	for num := 0; num < runs; num++ {

		// Prepare command to send and message on success to receive.
		command := fmt.Sprintf("%d APPEND INBOX {%d}", num, appendMsgSize)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send APPEND commmand to server.
		err := dovecotC.Send(command)
		if err != nil {
			log.Fatalf("[evaluation.TestAppend] %d: Failed during sending APPEND command: %s\n", num, err.Error())
		}

		// Receive answer to APPEND request.
		answer, err := dovecotC.Receive()
		if err != nil {
			log.Fatalf("[evaluation.TestAppend] %d: Error receiving response to APPEND: %s\n", num, err.Error())
		}

		if answer != "+ OK" {
			log.Fatalf("[evaluation.TestAppend] %d: Did not receive continuation command from server: %s\n", num, answer)
		}

		// Send mail message without additional newline.
		_, err = fmt.Fprintf(dovecotC.Conn, "%s\n", appendMsg)
		if err != nil {
			log.Fatalf("[evaluation.TestAppend] %d: Sending mail message to server failed with: %s\n", num, err.Error())
		}

		// Receive answer to message transfer.
		answer, err = dovecotC.Receive()
		if err != nil {
			log.Fatalf("[evaluation.TestAppend] %d: Error during receiving response to APPEND: %s\n", num, err.Error())
		}

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		if strings.Contains(answer, "Append completed") != true {
			log.Fatalf("[evaluation.TestAppend] %d: Server responded incorrectly to APPEND command: %s\n", num, answer)
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Store result in buffer.
		results[num] = rtt

		// Append log line to file.
		dovecotLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	err = dovecotC.Send("z LOGOUT")
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Error receiving first part of LOGOUT response: %s\n", err.Error())
	}

	// Receive next line from server.
	nextAnswer, err = dovecotC.Receive()
	if err != nil {
		log.Fatalf("[evaluation.TestAppend] Error receiving second part of LOGOUT response: %s\n", err.Error())
	}

	answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

	if strings.Contains(answer, "Logging out") != true {
		log.Fatalf("[evaluation.TestAppend] Server responded incorrectly to LOGOUT: %s\n", answer)
	}

	// Calculate statistics and print them.
	sum = 0
	for _, result := range results {
		sum += result
	}

	msAvg = 0
	msAvg = (float64(sum) / float64(runs)) / float64(time.Millisecond)

	log.Printf("[evaluation.TestAppend] Done on Dovecot, sent %d messages, each took %f ms on average.", runs, msAvg)
}
