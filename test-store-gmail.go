package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"crypto/tls"

	"github.com/numbleroot/pluto-evaluation/config"
	"github.com/numbleroot/pluto/imap"
)

// Functions

func main() {

	// Make test config file location and number of messages
	// to send per test configurable.
	configFlag := flag.String("config", "test-config.toml", "Specify location of config file that describes test setup configuration.")
	runsFlag := flag.Int("runs", 100, "Specify how many times the command of this test is to be sent to server.")
	flag.Parse()

	runs := *runsFlag

	log.Printf("Testing STORE command on gmail...\n")

	// Read configuration from file.
	config, err := config.LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("Error loading config: %s\n", err.Error())
	}

	// Create connection string to connect to gmail.
	gmailIMAPAddr := fmt.Sprintf("%s:%s", config.Gmail.Server, config.Gmail.Port)

	log.Printf("Connecting to gmail...\n")

	// Connect to remote gmail system.
	gmailConn, err := tls.Dial("tcp", gmailIMAPAddr, nil)
	if err != nil {
		log.Fatalf("Was unable to connect to remote gmail server: %s\n", err.Error())
	}

	// Create a new connection struct based on it.
	gmailC := &imap.Connection{
		OutConn:   gmailConn,
		OutReader: bufio.NewReader(gmailConn),
	}

	// Consume mandatory IMAP greeting.
	_, err = gmailC.Receive(false)
	if err != nil {
		log.Fatalf("Error during receiving initial server greeting: %s\n", err.Error())
	}

	// Log in as first user.
	err = gmailC.Send(false, fmt.Sprintf("storeA LOGIN %s %s", config.Gmail.StoreTest.Name, config.Gmail.StoreTest.Password))
	if err != nil {
		log.Fatalf("Sending LOGIN to server failed with: %s\n", err.Error())
	}

	// Wait for success message.
	answer, err := gmailC.Receive(false)
	if err != nil {
		log.Fatalf("Error during LOGIN as user %s: %s\n", config.Gmail.StoreTest.Name, err.Error())
	}

	// Receive next line from server.
	nextAnswer, err := gmailC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving second part of LOGIN response: %s\n", err.Error())
	}

	answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

	if strings.Contains(answer, "storeA OK") != true {
		log.Fatalf("Server responded unexpectedly to LOGIN: %s\n", answer)
	}

	log.Printf("Logged in as '%s'.\n", config.Gmail.StoreTest.Name)

	// Select INBOX for all following commands.
	err = gmailC.Send(false, "storeB SELECT INBOX")
	if err != nil {
		log.Fatalf("Sending SELECT to server failed with: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = gmailC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving first part of SELECT response: %s\n", err.Error())
	}

	// As long as the IMAP command termination indicator
	// was not yet received, continue to append answers.
	for (strings.Contains(answer, "selected") != true) &&
		(strings.Contains(answer, "BAD") != true) &&
		(strings.Contains(answer, "NO") != true) {

		// Receive next line from distributor.
		nextAnswer, err := gmailC.Receive(false)
		if err != nil {
			log.Fatalf("Error receiving next part of SELECT response: %s\n", err.Error())
		}

		answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)
	}

	if strings.Contains(answer, "storeB OK") != true {
		log.Fatalf("Server responded unexpectedly to SELECT: %s\n", answer)
	}

	log.Printf("Selected INBOX for further commands.\n")

	// Take current time stamp and create log file name.
	logFileTime := time.Now()
	gmailLogFileName := fmt.Sprintf("results/gmail-store-%s.log", logFileTime.Format("2006-01-02-15-04-05"))

	// Attempt to create a test log file containing
	// measured test times for gmail system.
	gmailLogFile, err := os.Create(gmailLogFileName)
	if err != nil {
		log.Fatalf("Failed to create test log file '%s': %s\n", gmailLogFileName, err.Error())
	}

	// Sync to storage and close on any exit.
	defer gmailLogFile.Close()
	defer gmailLogFile.Sync()

	// Prepend file with meta information about this test.
	gmailLogFile.WriteString(fmt.Sprintf("Subject: STORE\nPlatform: gmail\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

	// Prepare buffer to append individual results to.
	results := make([]int64, runs)

	log.Printf("Running tests on gmail...\n")

	for num := 1; num <= runs; num++ {

		i := num - 1

		// Prepare command to send.
		command := fmt.Sprintf("store%d STORE %d +FLAGS.SILENT (\\Seen \\Deleted)", num, num)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send STORE commmand to server.
		err := gmailC.Send(false, command)
		if err != nil {
			log.Fatalf("%d: Failed during sending STORE command: %s\n", num, err.Error())
		}

		// Receive answer to STORE request.
		answer, err := gmailC.Receive(false)
		if err != nil {
			log.Fatalf("%d: Error receiving response to STORE: %s\n", num, err.Error())
		}

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		if strings.Contains(answer, "Success") != true {
			log.Fatalf("%d: Server responded unexpectedly to STORE command: %s\n", num, answer)
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Store result in buffer.
		results[i] = rtt

		// Append log line to file.
		gmailLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	err = gmailC.Send(false, "storeZ LOGOUT")
	if err != nil {
		log.Fatalf("Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = gmailC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving first part of LOGOUT response: %s\n", err.Error())
	}

	// Receive next line from server.
	nextAnswer, err = gmailC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving second part of LOGOUT response: %s\n", err.Error())
	}

	answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

	if strings.Contains(answer, "(Success)") != true {
		log.Fatalf("Server responded unexpectedly to LOGOUT: %s\n", answer)
	}

	// Calculate statistics and print them.
	var sum int64 = 0
	for _, result := range results {
		sum += result
	}

	msAvg := (float64(sum) / float64(runs)) / float64(time.Millisecond)

	log.Printf("Done on gmail, sent %d messages, each took %f ms on average.\n\n", runs, msAvg)
}
