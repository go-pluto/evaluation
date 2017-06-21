package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"math/rand"

	"github.com/numbleroot/pluto-evaluation/imap-benchmark/config"
	"github.com/numbleroot/pluto-evaluation/imap-benchmark/sessions"
	"github.com/numbleroot/pluto-evaluation/imap-benchmark/worker"
)

// Functions

func main() {

	// Parse the input flags.
	configFlag := flag.String("config", "test-config.toml", "Specify location of config file that describes test setup configuration.")
	userdbFlag := flag.String("userdb", "userdb.passwd", "Specify location of the user/password file.")
	flag.Parse()

	// Read configuration from file.
	config, err := config.LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Load users from userdb file.
	users, err := config.LoadUsers(*userdbFlag)
	if err != nil {
		log.Fatalf("Error loading users from '%s' file: %v", *userdbFlag, err)
	}

	// Check results folder existence and create
	// a log file for this run.
	logFile, err := config.CreateLog()
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()
	defer logFile.Sync()

	// Seed the random number generator.
	rand.Seed(config.Settings.Seed)

	// Create the buffered channels. Channel "jobs" is for each session,
	// channel "logger" for the logged parameters (e.g. response time).
	jobs := make(chan worker.Session, 100)
	logger := make(chan []string, 100)

	// Start the worker pool.
	for w := 1; w <= config.Settings.Threads; w++ {
		go worker.Worker(w, config, jobs, logger)
	}

	// Assign jobs sessions.
	for j := 1; j <= config.Settings.Sessions; j++ {

		// Randomly choose a user.
		i := rand.Intn(len(users))

		// Hand over the job to the worker.
		jobs <- worker.Session{
			User:     users[i].Username,
			Password: users[i].Password,
			ID:       j,
			Commands: sessions.GenerateSession(config.Session.Minlength, config.Session.Maxlength),
		}
	}

	log.Printf("Generated %d sessions", config.Settings.Sessions)

	// Close jobs channel to stop all worker routines.
	close(jobs)

	// Collect results and write them to disk.
	for a := 1; a <= config.Settings.Sessions; a++ {

		logline := <-logger
		log.Printf("Finished %s", logline[1])

		for i := 0; i < len(logline); i++ {
			logFile.WriteString(logline[i])
			logFile.WriteString("\n")
		}
	}
}
