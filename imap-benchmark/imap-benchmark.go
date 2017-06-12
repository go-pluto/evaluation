package main
import (
    "fmt"
    "math/rand"
    "./config"
    "./sessions"
    "./worker"
)

func main() {

    // Read configuration from file.
    // TODO: add command line argument feature
    config, err := config.LoadConfig("readme1.toml")
    if err != nil {
        fmt.Println(err)
    }

    // create the buffered channels. Channel "jobs" for the session, channel
    // "results" for the return value. "results" channel is probably not used.
    jobs := make(chan worker.Session, 100)
    results := make(chan int, 100)

    // start the worker pool
    for w := 1; w <= config.Settings.Threads; w++ {
        go worker.Worker(w, config, jobs, results)
    }

    // seed the random number generator
    rand.Seed(config.Settings.Seed)

    // assign jobs i.e. sessions
    for j := 1; j <= 3; j++ {
        // hand the session to the worker routines
        jobs <- worker.Session{User: "ulf", Commands: sessions.GenerateSession(config.Session.Length)}
    }

    // close the jobs channel to close stop all worker routines
    close(jobs)

    // collect the results, probably not necessary
    for a := 1; a <= 3; a++ {
        <-results
    }
}
