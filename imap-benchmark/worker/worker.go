package worker
import (
    "fmt"
    "../sessions"
    "../config"
    "log"
    "bufio"
    "github.com/numbleroot/pluto/imap"
)

type Session struct {
	User           string
	Commands       []sessions.IMAPcommand
}

func Worker(id int,config *config.Config, jobs chan Session, results chan<- int) {
    for job := range jobs {

        connection := dialServer("imap.gmail.com","993")

        conn := &imap.Connection{
            OutConn:   connection,
            OutReader: bufio.NewReader(connection),
        }

        login(conn, "user", "pw", id)

        for i := 0; i < len(job.Commands); i++ {
            identifier := fmt.Sprintf("%dX%d", id, i)
            var command string

            switch job.Commands[i].Command {
            case "CREATE":
                command = fmt.Sprintf("%s CREATE %dX%s", identifier, id, job.Commands[i].Arguments[0])
                sendSimpleCommand(conn, command)
                log.Printf("creating folder\n")
            case "DELETE":
                command = fmt.Sprintf("%s DELETE %dX%s", identifier, id, job.Commands[i].Arguments[0])
                sendSimpleCommand(conn, command)
                log.Printf("deleting folder\n")
            case "APPEND":
                command = fmt.Sprintf("%s APPEND %dX%s %s %s", identifier, id, job.Commands[i].Arguments[0], job.Commands[i].Arguments[1], job.Commands[i].Arguments[2])
                sendAppendCommand(conn, command, job.Commands[i].Arguments[3])
                log.Printf("appending message\n")
            case "SELECT":
                command = fmt.Sprintf("%s SELECT %dX%s", identifier, id, job.Commands[i].Arguments[0])
                sendSimpleCommand(conn, command)
                log.Printf("selecting folder\n")
            case "STORE":
                command = fmt.Sprintf("%s STORE %s FLAGS %s", identifier, job.Commands[i].Arguments[0], job.Commands[i].Arguments[1])
                sendSimpleCommand(conn, command)
                log.Printf("storing message\n")
            case "EXPUNGE":
                command = fmt.Sprintf("%s EXPUNGE", identifier)
                sendSimpleCommand(conn, command)
                log.Printf("running expunge\n")
            case "CLOSE":
                command = fmt.Sprintf("%s CLOSE", identifier)
                sendSimpleCommand(conn, command)
                log.Printf("closing folder\n")
            }
        }

        logout(conn, id)
        results <- 2
    }
}
