# IMAP Benchmark

A tool to generate IMAP traffic for [pluto](https://github.com/numbleroot/pluto), [Dovecot](https://www.dovecot.org/), and other IMAP services (like GMail).

## Traffic Generateration

The major difference to the previously introduced imap-evaluation is, that we now
support *IMAP Sessions*. Sessions are sequences of IMAP commands that are can
be executed consecutively. The commands are *more or less* reasonable.

For the moment we only focus on **write** commands like:
  * CREATE
  * DELETE
  * APPEND
  * STORE
  * EXPUNGE    

## Setup

To install imap-benchmark, please run

```
$ go get -u github.com/numbleroot/pluto-evaluation
```

and go in the imap-benchmark folder of the cloned repository. For the moment
it is recommended to create a `results` folder for the logfiles by:

```
mkdir results
```

Next, modify the config file `test-config.toml` and the user data base `userdb.passwd`.

## Using

You can start benchmarking an imap service by running the imap-benchmark.go file.

```
$ go run imap-benchmark.go
```

Alternatively, you can provide paths for the config/userdb files:

```
$ go run imap-benchmark.go --config=/var/config.toml --userdb=/var/private.passwd
```

## Logging

All response times are collected in a logfile in the `results` folder.


## License

This project is [GPLv3](https://github.com/numbleroot/pluto-evaluation/blob/master/LICENSE) licensed.
