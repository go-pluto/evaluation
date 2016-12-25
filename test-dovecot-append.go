package main

import (
	"fmt"
	"log"

	"github.com/emersion/go-imap/client"
	"github.com/numbleroot/pluto-evaluation/config"
)

// Functions

func main() {

	var err error
	var imapClient *client.Client

	// Read configuration from file.
	config, err := config.LoadConfig("test-config-aws.toml")
	if err != nil {
		log.Fatalf("[evaluation.TestDovecotAppend] Error loading config: %s\n", err.Error())
	}

	// Create connection string to connect to.
	imapAddr := fmt.Sprintf("%s:%s", config.Dovecot.IP, config.Dovecot.Port)

	// Connect to remote Dovecot system.
	if config.Dovecot.TLS {
		imapClient, err = client.DialTLS(imapAddr, nil)
	} else {
		imapClient, err = client.Dial(imapAddr)
	}

	if err != nil {
		log.Fatalf("[evaluation.TestDovecotAppend] Was unable to connect to remote IMAP server: %s\n", err.Error())
	}

	// Log in as first user.
	err = imapClient.Login("test", "test")
	if err != nil {
		log.Fatalf("[evaluation.TestDovecotAppend] Failed to login as 'test': %s\n", err.Error())
	}

	// Log out on function exit.
	defer imapClient.Logout()

	// Select INBOX as mailbox.
	inbox, err := imapClient.Select("INBOX", false)
	if err != nil {
		log.Fatalf("[evaluation.TestDovecotAppend] Error during selecting 'INBOX': %s\n", err.Error())
	}

	log.Printf("inbox: %#v\n", inbox)

	// For each mail to append:
	// * take current time stamp A
	// * prepare log line
	// * send mail to remote system
	// * wait for response
	// * log reponse time stamp B
	// * calculate rtt = B - A
	// * finish log line and append to test log

	// Calculate statistics and print them.

	// Close log file and exit.
}
