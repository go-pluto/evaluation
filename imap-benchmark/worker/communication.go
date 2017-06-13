package worker

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/numbleroot/pluto/imap"
)

// Functions

// dialServer creates a tls secured connection to
// the server with the given hostname and port.
func dialServer(hostname string, port string) *tls.Conn {

	server := fmt.Sprintf("%s:%s", hostname, port)

	tlsConfig := &tls.Config{
		// Unfortunately, we currently need to accept this.
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", server, tlsConfig)
	if err != nil {
		log.Fatalf("Was unable to connect to remote server: %s\n", err.Error())
	}

	return conn
}

// login sends a login command with the given
// username/password combination on the given
// tls connection.
func login(conn *imap.Connection, username string, password string, id int) {

	// Consume mandatory IMAP greeting.
	_, err := conn.Receive(false)
	if err != nil {
		log.Fatalf("Error during receiving initial server greeting: %s\n", err.Error())
	}

	err = conn.Send(false, fmt.Sprintf("%dX LOGIN %s %s", id, username, password))
	if err != nil {
		log.Fatalf("Sending LOGIN to server failed with: %s\n", err.Error())
	}

	// Wait for success message.
	answer, err := conn.Receive(false)
	if err != nil {
		log.Fatalf("Error during LOGIN as user: %s", err.Error())
		log.Fatalf("received: %s", answer)
	}

	for strings.Contains(answer, fmt.Sprintf("%dX OK", id)) != true {
		nextAnswer, err := conn.Receive(false)
		if err != nil {
			log.Fatalf("Error during receiving: %s\n", err.Error())
		}
		answer = nextAnswer
	}

}

// sendSimpleCommand sends an IMAP command string
// on the given connection "con". The time between
// the send of the message and the receive of
// the imap confirmation will be counted and returned.
func sendSimpleCommand(conn *imap.Connection, command string) int64 {

	timeStart := time.Now().UnixNano()

	err := conn.Send(false, command)
	if err != nil {
		log.Fatalf("Error during sending: %s\n", err.Error())
	}

	answer, err := conn.Receive(false)
	if err != nil {
		log.Fatalf("Error during receiving: %s\n", err.Error())
	}

	for strings.HasPrefix(answer, strings.Split(command, " ")[0]) != true {
		nextAnswer, err := conn.Receive(false)
		if err != nil {
			log.Fatalf("Error during receiving: %s\n", err.Error())
		}

		answer = nextAnswer
	}

	timeEnd := time.Now().UnixNano()

	if strings.Contains(answer, "OK") != true {
		log.Printf("Server responded unexpectedly to command: %s\n", command)
		log.Printf("Answer: %s\n", answer)
	}
	return (timeEnd - timeStart)
}

// sendAppendCommand sends an IMAP command string
// that contains an APPEND command on the given
// connection "con". The time between the send
// of the message and the receive of the imap confirmation
// will be counted and returned.
func sendAppendCommand(conn *imap.Connection, command string, literal string) int64 {

	timeStart := time.Now().UnixNano()

	err := conn.Send(false, command)
	if err != nil {
		log.Fatalf("Error during sending: %s\n", err.Error())
	}

	answer, err := conn.Receive(false)
	if err != nil {
		log.Fatalf("Error during receiving: %s\n", err.Error())
	}

	// TODO: check validity of the server's response
	// if answer != "+ go ahead" {
	// 	log.Fatalf("Did not receive continuation command from server\n")
	// }

	// Send mail message without additional newline.
	appendMsg := bytes.NewBufferString(literal)
	_, err = fmt.Fprintf(conn.OutConn, "%s\r\n", appendMsg)
	if err != nil {
		log.Fatalf("Sending mail message to server failed with: %s\n", err.Error())
	}

	// Receive answer to message transfer.
	answer, err = conn.Receive(false)
	if err != nil {
		log.Fatalf("Error during receiving response to APPEND: %s\n", err.Error())
	}

	timeEnd := time.Now().UnixNano()

	if strings.Contains(answer, "OK") != true {
		log.Printf("Server responded unexpectedly to command: %s\n", command)
		log.Printf("Answer: %s\n", answer)
	}

	return (timeEnd - timeStart)
}

// logout sends a loglout command to the server
func logout(conn *imap.Connection, id int) {
	// Log out.
	err := conn.Send(false, fmt.Sprintf("%dZ LOGOUT", id))
	if err != nil {
		log.Fatalf("Error during LOGOUT: %s\n", err.Error())
	}

	// Receive first part of answer.
	answer, err := conn.Receive(false)
	if err != nil {
		log.Fatalf("Error receiving first part of LOGOUT response: %s\n", err.Error())
	}

	for strings.Contains(answer, fmt.Sprintf("%dZ", id)) != true {
		nextAnswer, err := conn.Receive(false)
		if err != nil {
			log.Fatalf("Error during LOGOUT: %s\n", err.Error())
		}

		answer = nextAnswer
	}

	// Receive next line from server.
	// nextAnswer, err := conn.Receive(false)
	// if err != nil {
	//     log.Fatalf("Error receiving second part of LOGOUT response: %s\n", err.Error())
	// }
	//
	// answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)
	//
	// if strings.Contains(answer, "(Success)") != true {
	//     log.Fatalf("Server responded unexpectedly to LOGOUT: %s\n", answer)
	// }
}
