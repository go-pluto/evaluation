package conn

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

// Structs

// Conn carries only needed elements for evaluation
// scripts to base receive and send methods on.
type Conn struct {
	Conn   net.Conn
	Reader *bufio.Reader
}

// Functions

// NewConn initializes and returns a new connection
// based on above struct.
func NewConn(c net.Conn) *Conn {

	return &Conn{
		Conn:   c,
		Reader: bufio.NewReader(c),
	}
}

// Receive returns the next network line delimited
// by IMAP RFC defined newline character.
func (c *Conn) Receive() (string, error) {

	var err error

	// Initial value for received message in order
	// to skip past the mandatory ping message.
	text := "> ping <\r\n"

	for text == "> ping <\r\n" {

		text, err = c.Reader.ReadString('\n')
		if err != nil {

			if err.Error() == "EOF" {
				log.Printf("[conn.Receive] Node at %s disconnected...\n", c.Conn.RemoteAddr())
			}

			break
		}
	}

	// If an error happened, return it.
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(text, "\r\n"), nil
}

// Send in turn sends a textual message to the other
// side of the connection.
func (c *Conn) Send(text string) error {

	if _, err := fmt.Fprintf(c.Conn, "%s\r\n", text); err != nil {
		return err
	}

	return nil
}
