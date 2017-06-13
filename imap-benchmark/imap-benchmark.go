package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/numbleroot/pluto-evaluation/imap-benchmark/config"
	"github.com/numbleroot/pluto-evaluation/imap-benchmark/sessions"
	"github.com/numbleroot/pluto-evaluation/imap-benchmark/worker"
)

// Structs

// Represents a user by the username and password.
type User struct {
	Username string
	Password string
}

// Functions

func main() {

	// Parse the input flags.
	configFlag := flag.String("config", "test-config.toml", "Specify location of config file that describes test setup configuration.")
	userdbFlag := flag.String("userdb", "userdb.passwd", "Specify location of the user/password file.")
	flag.Parse()

	// Read configuration from file.
	config, err := config.LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("Error loading config: %s\n", err.Error())
		fmt.Println(err)
	}

	// Open userdb file.
	userdb, err := os.Open(*userdbFlag)
	if err != nil {
		log.Fatal("Error loading userdb: %s\n", err.Error())
	}
	defer userdb.Close()

	var users []User

	// Scan line by line.
	scanner := bufio.NewScanner(userdb)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), ":")
		users = append(users, User{line[0], strings.TrimPrefix(line[1], "{plain}")})
	}

	// Check for errors while scanning.
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	// Create the buffered channels. Channel "jobs" for the session, channel
	// "logger" for the logged parameters (e.g. response time).
	jobs := make(chan worker.Session, 100)
	logger := make(chan []string, 100)

	// Start the worker pool.
	for w := 1; w <= config.Settings.Threads; w++ {
		go worker.Worker(w, config, jobs, logger)
	}

	// Seed the random number generator.
	rand.Seed(config.Settings.Seed)

	// Assign jobs i.e. sessions.
	for j := 1; j <= config.Settings.Sessions; j++ {
		// Random the user
		userIndex := rand.Intn(len(users))

		// Hand over the job to the worker.
		jobs <- worker.Session{
			User:     users[userIndex].Username,
			Password: users[userIndex].Password,
			ID:       j,
			Commands: sessions.GenerateSession(config.Session.Minlength, config.Session.Maxlength)}
	}

	log.Printf("Generated %d Sessions", config.Settings.Sessions)

	// Close the jobs channel to stop all worker routines
	close(jobs)

	// Prepare the log file.
	logFileTime := time.Now()
	logFileName := fmt.Sprintf("results/%s.log", logFileTime.Format("2006-01-02-15-04-05"))

	logFile, err := os.Create(logFileName)
	if err != nil {
		log.Fatalf("Failed to create log file '%s': %s\n", logFileName, err.Error())
	}

	defer logFile.Close()
	defer logFile.Sync()

	// Collect the results and write them to disk.
	for a := 1; a <= config.Settings.Sessions; a++ {
		logline := <-logger
		log.Printf("Finished %s", logline[1])

		for i := 0; i < len(logline); i++ {
			logFile.WriteString(fmt.Sprintf("%s\n", logline[i]))
		}
	}
}
