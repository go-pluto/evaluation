package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
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

	log.Printf("[evaluation.TestAppendConcurrent] Testing APPEND command concurrently on pluto and Dovecot...\n")

	// Make number of messages to send per test configurable.
	runsFlag := flag.Int("runs", 100, "Specify how many times the command of this test is to be sent to server.")
	flag.Parse()

	runs := *runsFlag

	// Read configuration from file.
	config, err := config.LoadConfig("test-config-aws.toml")
	if err != nil {
		log.Fatalf("[evaluation.TestAppendConcurrent] Error loading config: %s\n", err.Error())
	}

	// Save number of concurrent tests for later use.
	numTests := len(config.Pluto.ConcurrentTest.User)

	// We need a wait group to be able to wait for
	// running goroutines to finish.
	wg := new(sync.WaitGroup)

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
		log.Fatalf("[evaluation.TestAppendConcurrent] Failed to load cert file: %s\n", err.Error())
	}

	// Append certificate to test client's root CA pool.
	if ok := plutoTLSConfig.RootCAs.AppendCertsFromPEM(plutoRootCert); !ok {
		log.Fatalf("[evaluation.TestAppendConcurrent] Failed to append cert.\n")
	}

	// Create connection string to connect to pluto and Dovecot.
	plutoIMAPAddr := fmt.Sprintf("%s:%s", config.Pluto.IP, config.Pluto.Port)
	dovecotIMAPAddr := fmt.Sprintf("%s:%s", config.Dovecot.IP, config.Dovecot.Port)

	// Take current time.
	logFileTime := time.Now()

	// Define a log folder for pluto and create it.
	plutoLogFolder := fmt.Sprintf("results/pluto-append-concurrent-%s", logFileTime.Format("2006-01-02-15-04-05"))

	err = os.Mkdir(plutoLogFolder, (os.ModeDir | 0700))
	if err != nil {
		log.Fatalf("[evaluation.TestAppendConcurrent] Failed to create folder: %s\n", err.Error())
	}

	// Define a log folder for Dovecot and create it.
	dovecotLogFolder := fmt.Sprintf("results/dovecot-append-concurrent-%s", logFileTime.Format("2006-01-02-15-04-05"))

	err = os.Mkdir(dovecotLogFolder, (os.ModeDir | 0700))
	if err != nil {
		log.Fatalf("[evaluation.TestAppendConcurrent] Failed to create folder: %s\n", err.Error())
	}

	// Prepare message to append.
	appendMsg := bytes.NewBufferString(messages.Msg01)
	appendMsgSize := appendMsg.Len()

	log.Printf("[evaluation.TestAppendConcurrent] Connecting %d times to pluto...\n", numTests)

	for connNum := 0; connNum < numTests; connNum++ {

		// Increment wait group counter.
		wg.Add(1)

		go func(connNum int) {

			// On exit, let this goroutine signal to wait
			// group that it has finished.
			defer wg.Done()

			// Connect to remote pluto system.
			plutoConn, err := tls.Dial("tcp", plutoIMAPAddr, plutoTLSConfig)
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Was unable to connect to remote pluto server: %s\n", err.Error())
			}

			// Create connection based on it.
			plutoC := conn.NewConn(plutoConn)

			// Consume mandatory IMAP greeting.
			_, err = plutoC.Receive()
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Error during receiving initial server greeting: %s\n", err.Error())
			}

			// Log in as first user.
			err = plutoC.Send(fmt.Sprintf("appendA LOGIN %s %s", config.Pluto.ConcurrentTest.User[connNum].Name, config.Pluto.ConcurrentTest.User[connNum].Password))
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Sending LOGIN to server failed with: %s\n", err.Error())
			}

			// Wait for success message.
			answer, err := plutoC.Receive()
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Error during LOGIN as user %s: %s\n", config.Pluto.ConcurrentTest.User[connNum].Name, err.Error())
			}

			if strings.HasPrefix(answer, "appendA OK") != true {
				log.Fatalf("[evaluation.TestAppendConcurrent] Server responded incorrectly to LOGIN: %s\n", answer)
			}

			log.Printf("[evaluation.TestAppendConcurrent] Logged in as '%s'.\n", config.Pluto.ConcurrentTest.User[connNum].Name)

			// Define an individual test log file name.
			plutoLogFileName := fmt.Sprintf("%s/conn-%d.log", plutoLogFolder, connNum)

			// Attempt to create a test log file containing
			// measured test times for pluto system.
			plutoLogFile, err := os.Create(plutoLogFileName)
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Failed to create test log file '%s': %s\n", plutoLogFileName, err.Error())
			}

			// Sync to storage and close on any exit.
			defer plutoLogFile.Close()
			defer plutoLogFile.Sync()

			// Prepend file with meta information about this test.
			plutoLogFile.WriteString(fmt.Sprintf("Subject: Concurrent APPEND\nPlatform: pluto\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

			for num := 1; num <= runs; num++ {

				// Prepare command to send.
				command := fmt.Sprintf("append%d APPEND INBOX {%d}", num, appendMsgSize)

				// Take current time stamp.
				timeStart := time.Now().UnixNano()

				// Send APPEND commmand to server.
				err := plutoC.Send(command)
				if err != nil {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Failed during sending APPEND command: %s\n", num, err.Error())
				}

				// Receive answer to APPEND request.
				answer, err := plutoC.Receive()
				if err != nil {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Error receiving response to APPEND: %s\n", num, err.Error())
				}

				if answer != "+ Ready for literal data" {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Did not receive continuation command from server: %s\n", num, answer)
				}

				// Send mail message without additional newline.
				_, err = fmt.Fprintf(plutoC.Conn, "%s", appendMsg)
				if err != nil {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Sending mail message to server failed with: %s\n", num, err.Error())
				}

				// Receive answer to message transfer.
				answer, err = plutoC.Receive()
				if err != nil {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Error during receiving response to APPEND: %s\n", num, err.Error())
				}

				// Take time stamp after function execution.
				timeEnd := time.Now().UnixNano()

				if strings.Contains(answer, "APPEND completed") != true {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Server responded incorrectly to APPEND command: %s\n", num, answer)
				}

				// Calculate round-trip time.
				rtt := timeEnd - timeStart

				// Append log line to file.
				plutoLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
			}

			// Log out.
			err = plutoC.Send("appendZ LOGOUT")
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Error during LOGOUT: %s\n", err.Error())
			}

			// Receive first part of answer.
			answer, err = plutoC.Receive()
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Error receiving first part of LOGOUT response: %s\n", err.Error())
			}

			// Receive next line from server.
			nextAnswer, err := plutoC.Receive()
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Error receiving second part of LOGOUT response: %s\n", err.Error())
			}

			answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

			if strings.Contains(answer, "LOGOUT completed") != true {
				log.Fatalf("[evaluation.TestAppendConcurrent] Server responded incorrectly to LOGOUT: %s\n", answer)
			}

		}(connNum)
	}

	// Wait for all routines on pluto to have finished.
	wg.Wait()

	log.Printf("[evaluation.TestAppendConcurrent] Done on pluto, sent %d * %d = %d messages.\n\n", numTests, runs, (numTests * runs))

	// Run tests on Dovecot.
	log.Printf("[evaluation.TestAppendConcurrent] Connecting %d times to Dovecot...\n", numTests)

	for connNum := 0; connNum < numTests; connNum++ {

		// Increment wait group counter.
		wg.Add(1)

		go func() {

			// On exit, let this goroutine signal to wait
			// group that it has finished.
			defer wg.Done()

			// Connect to remote Dovecot system.
			dovecotConn, err := net.Dial("tcp", dovecotIMAPAddr)
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Was unable to connect to remote Dovecot server: %s\n", err.Error())
			}

			// Create connection based on it.
			dovecotC := conn.NewConn(dovecotConn)

			// Consume mandatory IMAP greeting.
			_, err = dovecotC.Receive()
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Error during receiving initial server greeting: %s\n", err.Error())
			}

			// Log in as first user.
			err = dovecotC.Send(fmt.Sprintf("appendA LOGIN %s %s", config.Dovecot.ConcurrentTest.User[connNum].Name, config.Dovecot.ConcurrentTest.User[connNum].Password))
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Sending LOGIN to server failed with: %s\n", err.Error())
			}

			// Wait for success message.
			answer, err := dovecotC.Receive()
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Error during LOGIN as user %s: %s\n", config.Dovecot.ConcurrentTest.User[connNum].Name, err.Error())
			}

			if strings.HasPrefix(answer, "appendA OK") != true {
				log.Fatalf("[evaluation.TestAppendConcurrent] Server responded incorrectly to LOGIN: %s\n", answer)
			}

			log.Printf("[evaluation.TestAppendConcurrent] Logged in as '%s'.\n", config.Dovecot.ConcurrentTest.User[connNum].Name)

			// Define an individual test log file name.
			dovecotLogFileName := fmt.Sprintf("%s/conn-%d.log", dovecotLogFolder, connNum)

			// Attempt to create a test log file containing
			// measured test times for Dovecot system.
			dovecotLogFile, err := os.Create(dovecotLogFileName)
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Failed to create test log file '%s': %s\n", dovecotLogFileName, err.Error())
			}

			// Sync to storage and close on any exit.
			defer dovecotLogFile.Close()
			defer dovecotLogFile.Sync()

			// Prepend file with meta information about this test.
			dovecotLogFile.WriteString(fmt.Sprintf("Subject: Concurrent APPEND\nPlatform: Dovecot\nDate: %s\n-----\n", logFileTime.Format("2006-01-02-15-04-05")))

			for num := 1; num <= runs; num++ {

				// Prepare command to send.
				command := fmt.Sprintf("append%d APPEND INBOX {%d}", num, appendMsgSize)

				// Take current time stamp.
				timeStart := time.Now().UnixNano()

				// Send APPEND commmand to server.
				err := dovecotC.Send(command)
				if err != nil {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Failed during sending APPEND command: %s\n", num, err.Error())
				}

				// Receive answer to APPEND request.
				answer, err := dovecotC.Receive()
				if err != nil {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Error receiving response to APPEND: %s\n", num, err.Error())
				}

				if answer != "+ OK" {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Did not receive continuation command from server: %s\n", num, answer)
				}

				// Send mail message.
				_, err = fmt.Fprintf(dovecotC.Conn, "%s\n", appendMsg)
				if err != nil {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Sending mail message to server failed with: %s\n", num, err.Error())
				}

				// Receive answer to message transfer.
				answer, err = dovecotC.Receive()
				if err != nil {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Error during receiving response to APPEND: %s\n", num, err.Error())
				}

				// Take time stamp after function execution.
				timeEnd := time.Now().UnixNano()

				if strings.Contains(answer, "Append completed") != true {
					log.Fatalf("[evaluation.TestAppendConcurrent] %d: Server responded incorrectly to APPEND command: %s\n", num, answer)
				}

				// Calculate round-trip time.
				rtt := timeEnd - timeStart

				// Append log line to file.
				dovecotLogFile.WriteString(fmt.Sprintf("%d, %d\n", num, rtt))
			}

			// Log out.
			err = dovecotC.Send("appendZ LOGOUT")
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Error during LOGOUT: %s\n", err.Error())
			}

			// Receive first part of answer.
			answer, err = dovecotC.Receive()
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Error receiving first part of LOGOUT response: %s\n", err.Error())
			}

			// Receive next line from server.
			nextAnswer, err := dovecotC.Receive()
			if err != nil {
				log.Fatalf("[evaluation.TestAppendConcurrent] Error receiving second part of LOGOUT response: %s\n", err.Error())
			}

			answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

			if strings.Contains(answer, "Logging out") != true {
				log.Fatalf("[evaluation.TestAppendConcurrent] Server responded incorrectly to LOGOUT: %s\n", answer)
			}

		}()
	}

	// Wait for all routines on Dovecot to have finished.
	wg.Wait()

	log.Printf("[evaluation.TestAppendConcurrent] Done on Dovecot, sent %d * %d = %d messages.\n\n", numTests, runs, (numTests * runs))
}
