package worker

import (
	"bufio"
	"fmt"

	"github.com/numbleroot/pluto-evaluation/imap-benchmark/config"
	"github.com/numbleroot/pluto-evaluation/imap-benchmark/sessions"
	"github.com/numbleroot/pluto/imap"
)

// Structs

// Session contains the user's credentials, an identifier for the
// session and a sequence of IMAP commands that has been generated
// by the sessions package.
type Session struct {
	User     string
	Password string
	ID       int
	Commands []sessions.IMAPcommand
}

// Functions

// Worker is the routine that sends the commands of the session
// to the server. The output will be logged and written in
// the logger channel.
func Worker(id int, config *config.Config, jobs chan Session, logger chan<- []string) {

	for job := range jobs {

		var output []string

		output = append(output, "########################")
		output = append(output, fmt.Sprintf("Session: %d", job.ID))
		output = append(output, fmt.Sprintf("User: %s", job.User))
		output = append(output, fmt.Sprintf("Password: %s", job.Password))
		output = append(output, "---- starting commands ----")

		// Connect to remote server.
		connection := dialServer(config.Server.Hostname, config.Server.Port)

		conn := &imap.Connection{
			OutConn:   connection,
			OutReader: bufio.NewReader(connection),
		}

		login(conn, job.User, job.Password, id)

		for i := 0; i < len(job.Commands); i++ {

			switch job.Commands[i].Command {

			case "CREATE":
				command := fmt.Sprintf("%dX%d CREATE %dX%s", id, i, id, job.Commands[i].Arguments[0])
				respTime := sendSimpleCommand(conn, command)
				output = append(output, fmt.Sprintf("CREATE %d", respTime))
				// log.Println("creating folder")

			case "DELETE":
				command := fmt.Sprintf("%dX%d DELETE %dX%s", id, i, id, job.Commands[i].Arguments[0])
				respTime := sendSimpleCommand(conn, command)
				output = append(output, fmt.Sprintf("DELETE %d", respTime))
				// log.Println("deleting folder")

			case "APPEND":
				command := fmt.Sprintf("%dX%d APPEND %dX%s %s %s", id, i, id, job.Commands[i].Arguments[0], job.Commands[i].Arguments[1], job.Commands[i].Arguments[2])
				respTime := sendAppendCommand(conn, command, job.Commands[i].Arguments[3])
				output = append(output, fmt.Sprintf("APPEND %d", respTime))
				// log.Println("appending message")

			case "SELECT":
				command := fmt.Sprintf("%dX%d SELECT %dX%s", id, i, id, job.Commands[i].Arguments[0])
				respTime := sendSimpleCommand(conn, command)
				output = append(output, fmt.Sprintf("SELECT %d", respTime))
				// log.Println("selecting folder")

			case "STORE":
				command := fmt.Sprintf("%dX%d STORE %s FLAGS %s", id, i, job.Commands[i].Arguments[0], job.Commands[i].Arguments[1])
				respTime := sendSimpleCommand(conn, command)
				output = append(output, fmt.Sprintf("STORE %d", respTime))
				// log.Println("storing message")

			case "EXPUNGE":
				command := fmt.Sprintf("%dX%d EXPUNGE", id, i)
				respTime := sendSimpleCommand(conn, command)
				output = append(output, fmt.Sprintf("EXPUNGE %d", respTime))
				// log.Println("running expunge")

			case "CLOSE":
				command := fmt.Sprintf("%dX%d CLOSE", id, i)
				respTime := sendSimpleCommand(conn, command)
				output = append(output, fmt.Sprintf("CLOSE %d", respTime))
				// log.Println("closing folder")
			}
		}

		output = append(output, "########################")

		logout(conn, id)

		logger <- output
	}
}
