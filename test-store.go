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

	"github.com/go-pluto/evaluation/config"
	"github.com/go-pluto/evaluation/utils"
	"github.com/go-pluto/pluto/imap"
)

// Functions

func main() {

	// Make test config file location and number of messages
	// to send per test configurable.
	configFlag := flag.String("config", "test-config.toml", "Specify location of config file that describes test setup configuration.")
	runsFlag := flag.Int("runs", 100, "Specify how many times the command of this test is to be sent to server.")
	flag.Parse()

	runs := *runsFlag

	log.Printf("Testing STORE command on pluto and Dovecot...\n")

	// Read configuration from file.
	config, err := config.LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("Error loading config: %s\n", err.Error())
	}

	// Create needed TLS configs with correct certificates.
	plutoTLSConfig, dovecotTLSConfig, err := utils.InitTLSConfigs(config)
	if err != nil {
		log.Fatalf("Error loading TLS configs for pluto and Dovecot: %s\n", err.Error())
	}

	// Create connection string to connect to pluto and Dovecot.
	plutoIMAPAddr := fmt.Sprintf("%s:%s", config.Pluto.IP, config.Pluto.Port)
	dovecotIMAPAddr := fmt.Sprintf("%s:%s", config.Dovecot.IP, config.Dovecot.Port)

	log.Printf("Connecting to pluto...\n")

	// Connect to remote pluto system.
	plutoConn, err := tls.Dial("tcp", plutoIMAPAddr, plutoTLSConfig)
	if err != nil {
		log.Fatalf("Was unable to connect to remote pluto server: %s\n", err.Error())
	}

	// Create a new connection struct based on it.
	plutoC := &imap.Connection{
		OutConn:   plutoConn,
		OutReader: bufio.NewReader(plutoConn),
	}

	// Consume mandatory IMAP greeting.
	_, err = plutoC.Receive(false)
	if err != nil {
		log.Fatalf("Error during receiving initial server greeting: %s\n", err.Error())
	}

	// Log in as first user.
	err = plutoC.Send(false, fmt.Sprintf("storeA LOGIN %s %s", config.Pluto.StoreTest.Name, config.Pluto.StoreTest.Password))
	if err != nil {
		log.Fatalf("Sending LOGIN to server failed with: %s\n", err.Error())
	}

	// Wait for success message.
	answer, err := plutoC.Receive(false)
	if err != nil {
		log.Fatalf("Error during LOGIN as user %s: %s\n", config.Pluto.StoreTest.Name, err.Error())
	}

	if strings.HasPrefix(answer, "storeA OK") != true {
		log.Fatalf("Server responded unexpectedly to LOGIN: %s\n", answer)
	}

	log.Printf("Logged in as '%s'.\n", config.Pluto.StoreTest.Name)

	// Select INBOX for all following commands.
	err = plutoC.Send(false, "storeB SELECT INBOX")
	if err != nil {
		log.Fatalf("Sending SELECT to server failed with: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = plutoC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving first part of SELECT response: %s\n", err.Error())
	}

	// As long as the IMAP command termination indicator
	// was not yet received, continue to append answers.
	for (strings.Contains(answer, "completed") != true) &&
		(strings.Contains(answer, "BAD") != true) &&
		(strings.Contains(answer, "NO") != true) {

		// Receive next line from distributor.
		nextAnswer, err := plutoC.Receive(false)
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
	plutoLogFileName := fmt.Sprintf("results/pluto-store-%s.log", logFileTime.Format("2006-01-02-15-04-05"))

	// Attempt to create a test log file containing
	// measured test times for pluto system.
	plutoLogFile, err := os.Create(plutoLogFileName)
	if err != nil {
		log.Fatalf("Failed to create test log file '%s': %s\n", plutoLogFileName, err.Error())
	}

	// Sync to storage and close on any exit.
	defer plutoLogFile.Close()
	defer plutoLogFile.Sync()

	// Prepend file with meta information about this test.
	plutoLogFile.WriteString(fmt.Sprintf("Subject: STORE\nPlatform: pluto\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

	// Prepare buffer to append individual results to.
	results := make([]int64, runs)

	log.Printf("Running tests on pluto...\n")

	for num := 1; num <= runs; num++ {

		i := num - 1

		// Prepare command to send.
		command := fmt.Sprintf("store%d STORE %d +FLAGS.SILENT (\\Seen \\Deleted)", num, num)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send STORE commmand to server.
		err := plutoC.Send(false, command)
		if err != nil {
			log.Fatalf("%d: Failed during sending STORE command: %s\n", num, err.Error())
		}

		// Receive answer to STORE request.
		answer, err := plutoC.Receive(false)
		if err != nil {
			log.Fatalf("%d: Error receiving response to STORE: %s\n", num, err.Error())
		}

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		if strings.Contains(answer, "STORE completed") != true {
			log.Fatalf("%d: Server responded unexpectedly to STORE command: %s\n", num, answer)
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Store result in buffer.
		results[i] = rtt

		// Append log line to file.
		plutoLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	err = plutoC.Send(false, "storeZ LOGOUT")
	if err != nil {
		log.Fatalf("Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = plutoC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving first part of LOGOUT response: %s\n", err.Error())
	}

	// Receive next line from server.
	nextAnswer, err := plutoC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving second part of LOGOUT response: %s\n", err.Error())
	}

	answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

	if strings.Contains(answer, "LOGOUT completed") != true {
		log.Fatalf("Server responded unexpectedly to LOGOUT: %s\n", answer)
	}

	// Calculate statistics and print them.
	var sum int64 = 0
	for _, result := range results {
		sum += result
	}

	msAvg := (float64(sum) / float64(runs)) / float64(time.Millisecond)

	log.Printf("Done on pluto, sent %d store instructions, each took %f ms on average.\n\n", runs, msAvg)

	// Run tests on Dovecot.
	log.Printf("Connecting to Dovecot...\n")

	// Connect to remote Dovecot system.
	dovecotConn, err := tls.Dial("tcp", dovecotIMAPAddr, dovecotTLSConfig)
	if err != nil {
		log.Fatalf("Was unable to connect to remote Dovecot server: %s\n", err.Error())
	}

	// Create a new connection struct based on it.
	dovecotC := &imap.Connection{
		OutConn:   dovecotConn,
		OutReader: bufio.NewReader(dovecotConn),
	}

	// Consume mandatory IMAP greeting.
	_, err = dovecotC.Receive(false)
	if err != nil {
		log.Fatalf("Error during receiving initial server greeting: %s\n", err.Error())
	}

	// Log in as first user.
	err = dovecotC.Send(false, fmt.Sprintf("storeA LOGIN %s %s", config.Dovecot.StoreTest.Name, config.Dovecot.StoreTest.Password))
	if err != nil {
		log.Fatalf("Sending LOGIN to server failed with: %s\n", err.Error())
	}

	// Wait for success message.
	answer, err = dovecotC.Receive(false)
	if err != nil {
		log.Fatalf("Error during LOGIN as user %s: %s\n", config.Dovecot.StoreTest.Name, err.Error())
	}

	if strings.HasPrefix(answer, "storeA OK") != true {
		log.Fatalf("Server responded unexpectedly to LOGIN: %s\n", answer)
	}

	log.Printf("Logged in as '%s'.\n", config.Dovecot.StoreTest.Name)

	// Select INBOX for all following commands.
	err = dovecotC.Send(false, "storeB SELECT INBOX")
	if err != nil {
		log.Fatalf("Sending SELECT to server failed with: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = dovecotC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving first part of SELECT response: %s\n", err.Error())
	}

	// As long as the IMAP command termination indicator
	// was not yet received, continue to append answers.
	for (strings.Contains(answer, "completed") != true) &&
		(strings.Contains(answer, "BAD") != true) &&
		(strings.Contains(answer, "NO") != true) {

		// Receive next line from distributor.
		nextAnswer, err := dovecotC.Receive(false)
		if err != nil {
			log.Fatalf("Error receiving next part of SELECT response: %s\n", err.Error())
		}

		answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)
	}

	if strings.Contains(answer, "storeB OK") != true {
		log.Fatalf("Server responded unexpectedly to SELECT: %s\n", answer)
	}

	log.Printf("Selected INBOX for further commands.\n")

	// Prepare log file name for Dovecot.
	dovecotLogFileName := fmt.Sprintf("results/dovecot-store-%s.log", logFileTime.Format("2006-01-02-15-04-05"))

	// Attempt to create a test log file containing
	// measured test times for Dovecot system.
	dovecotLogFile, err := os.Create(dovecotLogFileName)
	if err != nil {
		log.Fatalf("Failed to create test log file '%s': %s\n", dovecotLogFileName, err.Error())
	}

	// Sync to storage and close on any exit.
	defer dovecotLogFile.Close()
	defer dovecotLogFile.Sync()

	// Prepend file with meta information about this test.
	dovecotLogFile.WriteString(fmt.Sprintf("Subject: STORE\nPlatform: Dovecot\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

	log.Printf("Running tests on Dovecot...\n")

	// Reset results slice.
	results = make([]int64, runs)

	for num := 1; num <= runs; num++ {

		i := num - 1

		// Prepare command to send.
		command := fmt.Sprintf("store%d STORE %d +FLAGS.SILENT (\\Seen \\Deleted)", num, num)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send STORE commmand to server.
		err := dovecotC.Send(false, command)
		if err != nil {
			log.Fatalf("%d: Failed during sending STORE command: %s\n", num, err.Error())
		}

		// Receive answer to STORE request.
		answer, err := dovecotC.Receive(false)
		if err != nil {
			log.Fatalf("%d: Error receiving response to STORE: %s\n", num, err.Error())
		}

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		if strings.Contains(answer, "Store completed") != true {
			log.Fatalf("%d: Server responded unexpectedly to STORE command: %s\n", num, answer)
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Store result in buffer.
		results[i] = rtt

		// Append log line to file.
		dovecotLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	err = dovecotC.Send(false, "storeZ LOGOUT")
	if err != nil {
		log.Fatalf("Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err = dovecotC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving first part of LOGOUT response: %s\n", err.Error())
	}

	// Receive next line from server.
	nextAnswer, err = dovecotC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving second part of LOGOUT response: %s\n", err.Error())
	}

	answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

	if strings.Contains(answer, "Logging out") != true {
		log.Fatalf("Server responded unexpectedly to LOGOUT: %s\n", answer)
	}

	// Calculate statistics and print them.
	sum = 0
	for _, result := range results {
		sum += result
	}

	msAvg = 0
	msAvg = (float64(sum) / float64(runs)) / float64(time.Millisecond)

	log.Printf("Done on Dovecot, sent %d store instructions, each took %f ms on average.", runs, msAvg)
}
