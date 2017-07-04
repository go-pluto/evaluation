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

func PlutoTester(start chan struct{}, done chan struct{}, plutoC *imap.Connection, connNum int, runs int, plutoLogFolder string, logFileTime time.Time) {

	// Define an individual test log file name.
	plutoLogFileName := fmt.Sprintf("%s/conn-%03d.log", plutoLogFolder, connNum)

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
	plutoLogFile.WriteString(fmt.Sprintf("Subject: Concurrent DELETE\nPlatform: pluto\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

	// Wait for signal to start test.
	<-start

	// Execute supplied amount of tests agains pluto.
	for num := 1; num <= runs; num++ {

		// Prepare command to send.
		command := fmt.Sprintf("delete%d DELETE evaluation-mailbox-%d", num, num)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send DELETE commmand to server.
		err := plutoC.Send(false, command)
		if err != nil {
			log.Fatalf("%d: Failed during sending DELETE command: %s\n", num, err.Error())
		}

		// Receive answer to DELETE request.
		answer, err := plutoC.Receive(false)
		if err != nil {
			log.Fatalf("%d: Error receiving response to DELETE: %s\n", num, err.Error())
		}

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		if strings.Contains(answer, "DELETE completed") != true {
			log.Fatalf("%d: Server responded unexpectedly to DELETE command: %s\n", num, answer)
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Append log line to file.
		plutoLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	err = plutoC.Send(false, "deleteZ LOGOUT")
	if err != nil {
		log.Fatalf("Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err := plutoC.Receive(false)
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

	// Send done signal back.
	done <- struct{}{}
}

func DovecotTester(start chan struct{}, done chan struct{}, dovecotC *imap.Connection, connNum int, runs int, dovecotLogFolder string, logFileTime time.Time) {

	// Define an individual test log file name.
	dovecotLogFileName := fmt.Sprintf("%s/conn-%03d.log", dovecotLogFolder, connNum)

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
	dovecotLogFile.WriteString(fmt.Sprintf("Subject: Concurrent DELETE\nPlatform: Dovecot\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

	// Wait for signal to start test.
	<-start

	for num := 1; num <= runs; num++ {

		// Prepare command to send.
		command := fmt.Sprintf("delete%d DELETE evaluation-mailbox-%d", num, num)

		// Take current time stamp.
		timeStart := time.Now().UnixNano()

		// Send DELETE commmand to server.
		err := dovecotC.Send(false, command)
		if err != nil {
			log.Fatalf("%d: Failed during sending DELETE command: %s\n", num, err.Error())
		}

		// Receive answer to DELETE request.
		answer, err := dovecotC.Receive(false)
		if err != nil {
			log.Fatalf("%d: Error receiving response to DELETE: %s\n", num, err.Error())
		}

		// Take time stamp after function execution.
		timeEnd := time.Now().UnixNano()

		if strings.Contains(answer, "Delete completed") != true {
			log.Fatalf("%d: Server responded unexpectedly to DELETE command: %s\n", num, answer)
		}

		// Calculate round-trip time.
		rtt := timeEnd - timeStart

		// Append log line to file.
		dovecotLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
	}

	// Log out.
	err = dovecotC.Send(false, "deleteZ LOGOUT")
	if err != nil {
		log.Fatalf("Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err := dovecotC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving first part of LOGOUT response: %s\n", err.Error())
	}

	// Receive next line from server.
	nextAnswer, err := dovecotC.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving second part of LOGOUT response: %s\n", err.Error())
	}

	answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

	if strings.Contains(answer, "Logging out") != true {
		log.Fatalf("Server responded unexpectedly to LOGOUT: %s\n", answer)
	}

	// Send done signal back.
	done <- struct{}{}
}

func main() {

	// Make test config file location and number of messages
	// to send per test configurable.
	configFlag := flag.String("config", "test-config.toml", "Specify location of config file that describes test setup configuration.")
	runsFlag := flag.Int("runs", 100, "Specify how many times the command of this test is to be sent to server.")
	flag.Parse()

	runs := *runsFlag

	log.Printf("Testing DELETE command concurrently on pluto and Dovecot...\n\n")

	// Read configuration from file.
	config, err := config.LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("Error loading config: %s\n", err.Error())
	}

	// Save number of concurrent tests for later use.
	numTests := len(config.Pluto.ConcurrentTest.User)

	// Check that same amount of users is configured for
	// both systems, pluto and Dovecot.
	if numTests != len(config.Dovecot.ConcurrentTest.User) {
		log.Fatalf("Please configure an equal number of concurrent test users for pluto and Dovecot.\n")
	}

	// Prepare buffered channels to signal start and
	// finish over to involved testing routines.
	start := make(chan struct{}, numTests)
	done := make(chan struct{}, numTests)

	// Create needed TLS configs with correct certificates.
	plutoTLSConfig, dovecotTLSConfig, err := utils.InitTLSConfigs(config)
	if err != nil {
		log.Fatalf("Error loading TLS configs for pluto and Dovecot: %s\n", err.Error())
	}

	// Create connection string to connect to pluto and Dovecot.
	plutoIMAPAddr := fmt.Sprintf("%s:%s", config.Pluto.IP, config.Pluto.Port)
	dovecotIMAPAddr := fmt.Sprintf("%s:%s", config.Dovecot.IP, config.Dovecot.Port)

	// Take current time.
	logFileTime := time.Now()

	// Define a log folder for pluto and create it.
	plutoLogFolder := fmt.Sprintf("results/pluto-delete-concurrent-%s", logFileTime.Format("2006-01-02-15-04-05"))

	err = os.Mkdir(plutoLogFolder, (os.ModeDir | 0700))
	if err != nil {
		log.Fatalf("Failed to create folder: %s\n", err.Error())
	}

	// Define a log folder for Dovecot and create it.
	dovecotLogFolder := fmt.Sprintf("results/dovecot-delete-concurrent-%s", logFileTime.Format("2006-01-02-15-04-05"))

	err = os.Mkdir(dovecotLogFolder, (os.ModeDir | 0700))
	if err != nil {
		log.Fatalf("Failed to create folder: %s\n", err.Error())
	}

	log.Printf("Connecting %d times to pluto...\n", numTests)

	for connNum := 0; connNum < numTests; connNum++ {

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
		err = plutoC.Send(false, fmt.Sprintf("deleteA LOGIN %s %s", config.Pluto.ConcurrentTest.User[connNum].Name, config.Pluto.ConcurrentTest.User[connNum].Password))
		if err != nil {
			log.Fatalf("Sending LOGIN to server failed with: %s\n", err.Error())
		}

		// Wait for success message.
		answer, err := plutoC.Receive(false)
		if err != nil {
			log.Fatalf("Error during LOGIN as user %s: %s\n", config.Pluto.ConcurrentTest.User[connNum].Name, err.Error())
		}

		if strings.HasPrefix(answer, "deleteA OK") != true {
			log.Fatalf("Server responded unexpectedly to LOGIN: %s\n", answer)
		}

		log.Printf("Logged in as '%s'.\n", config.Pluto.ConcurrentTest.User[connNum].Name)

		// Dispatch to own goroutine.
		go PlutoTester(start, done, plutoC, connNum, runs, plutoLogFolder, logFileTime)
	}

	// Send start signal to ready routines.
	for signal := 0; signal < numTests; signal++ {
		start <- struct{}{}
	}

	// Wait for all done signals to come in.
	for signal := 0; signal < numTests; signal++ {
		<-done
	}

	log.Printf("Done on pluto, sent %d * %d = %d messages.\n\n", numTests, runs, (numTests * runs))

	// Run tests on Dovecot.
	log.Printf("Connecting %d times to Dovecot...\n", numTests)

	for connNum := 0; connNum < numTests; connNum++ {

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
		err = dovecotC.Send(false, fmt.Sprintf("deleteA LOGIN %s %s", config.Dovecot.ConcurrentTest.User[connNum].Name, config.Dovecot.ConcurrentTest.User[connNum].Password))
		if err != nil {
			log.Fatalf("Sending LOGIN to server failed with: %s\n", err.Error())
		}

		// Wait for success message.
		answer, err := dovecotC.Receive(false)
		if err != nil {
			log.Fatalf("Error during LOGIN as user %s: %s\n", config.Dovecot.ConcurrentTest.User[connNum].Name, err.Error())
		}

		if strings.HasPrefix(answer, "deleteA OK") != true {
			log.Fatalf("Server responded unexpectedly to LOGIN: %s\n", answer)
		}

		log.Printf("Logged in as '%s'.\n", config.Dovecot.ConcurrentTest.User[connNum].Name)

		// Dispatch into own goroutine.
		go DovecotTester(start, done, dovecotC, connNum, runs, dovecotLogFolder, logFileTime)
	}

	// Send start signal to ready routines.
	for signal := 0; signal < numTests; signal++ {
		start <- struct{}{}
	}

	// Wait for all done signals to come in.
	for signal := 0; signal < numTests; signal++ {
		<-done
	}

	log.Printf("Done on Dovecot, sent %d * %d = %d messages.\n\n", numTests, runs, (numTests * runs))
}
