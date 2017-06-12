package worker

import (
    "fmt"
    "log"
    "crypto/tls"
    "github.com/numbleroot/pluto/imap"
    "strings"
	"bytes"
)

func dialServer(hostname string, port string) *tls.Conn {
	server := fmt.Sprintf("%s:%s", hostname, port)

    log.Printf("Connecting to server...")

    conn, err := tls.Dial("tcp", server, nil)
    if err != nil {
        log.Fatalf("Was unable to connect to remote server: %s\n", err.Error())
    }

    return conn
}

func login(conn *imap.Connection, username string, password string, id int){
    // Consume mandatory IMAP greeting.
    _, err := conn.Receive(false)
    if err != nil {
        log.Fatalf("Error during receiving initial server greeting: %s\n", err.Error())
    }

    // Send LOGIN command.
    err = conn.Send(false, fmt.Sprintf("%dX LOGIN %s %s", id, username, password))
    if err != nil {
        log.Fatalf("Sending LOGIN to server failed with: %s\n", err.Error())
    }

    // Wait for success message.
    answer, err := conn.Receive(false)
    if err != nil {
        log.Fatalf("Error during LOGIN as user", err.Error())
    }

    // Receive next line from server.
    nextAnswer, err := conn.Receive(false)
    if err != nil {
        log.Fatalf("Error receiving second part of LOGIN response: %s\n", err.Error())
    }

    answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

    if strings.Contains(answer, fmt.Sprintf("%dX OK", id)) != true {
        log.Fatalf("Server responded unexpectedly to LOGIN: %s\n", answer)
    }

    log.Printf("Logged in\n")
}

func sendSimpleCommand(conn *imap.Connection, command string) {

    err := conn.Send(false, command)
    if err != nil {
        log.Fatalf("Error during sending: %s\n", err.Error())
    }

    answer, err := conn.Receive(false)
    if err != nil {
        log.Fatalf("Error during receiving: %s\n", err.Error())
    }

    for (strings.HasPrefix(answer, strings.Split(command, " ")[0]) != true) {
        nextAnswer, err := conn.Receive(false)
        if err != nil {
            log.Fatalf("Error during receiving: %s\n", err.Error())
        }

        answer = nextAnswer
    }

    if strings.Contains(answer, "OK") != true {
			log.Printf("Server responded unexpectedly to command: %s\n", command)
            log.Printf("Answer: %s\n", answer)
	}
}

func sendAppendCommand(conn *imap.Connection, command string, literal string) {
    err := conn.Send(false, command)
    if err != nil {
        log.Fatalf("Error during sending: %s\n", err.Error())
    }

    answer, err := conn.Receive(false)
    if err != nil {
        log.Fatalf("Error during receiving: %s\n", err.Error())
    }

    if answer != "+ go ahead" {
		log.Fatalf("Did not receive continuation command from server\n")
	}

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

    if strings.Contains(answer, "OK") != true {
            log.Printf("Server responded unexpectedly to command: %s\n", command)
            log.Printf("Answer: %s\n", answer)
    }
}

func logout(conn *imap.Connection, id int){
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

    // Receive next line from server.
    nextAnswer, err := conn.Receive(false)
    if err != nil {
        log.Fatalf("Error receiving second part of LOGOUT response: %s\n", err.Error())
    }

    answer = fmt.Sprintf("%s\r\n%s", answer, nextAnswer)

    if strings.Contains(answer, "(Success)") != true {
        log.Fatalf("Server responded unexpectedly to LOGOUT: %s\n", answer)
    }

    log.Printf("Logged out\n")
}
