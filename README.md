# Pluto evaluation

Collection of test scripts for evaluating [pluto](https://github.com/go-pluto/pluto)'s performance compared to [Dovecot](https://www.dovecot.org).


## Setup

First, run

```
$ make folders
```

and place involved certificates under `private/`.

Next, copy `test-config.toml.example` to `test-config.toml` and adjust it to your setup. This includes specifying IPs, ports, certificates and authentication information in order for your tests to run successfully. Afterwards, execute

```
$ make build
```

which will re-run `folders` target, compile all test scripts and the executable to plot the results.


## Testing

Now you can start testing. For this, choose a test to start with from the available executables and run it, e.g.

```
$ ./test-append -runs 1000
```

which will execute 1000 APPEND operations against first pluto and then Dovecot. Result logs will be placed in `results/`, containing meta-information and comma-separated pairs of msgID and completion time of that command in nanoseconds. A beginning of such a file might look like:

```
Subject: APPEND
Platform: pluto
Date: 2017-01-01-10-00-00
-----
1, 110422612
2, 98924309
3, 107112562
...
```


## Plotting

Finally, you can plot two corresponding test results against each other with every run of `plot-results`. For this, execute

```
$ ./plot-results -fileOne results/pluto-append-2017-01-01-10-00-00.log -fileTwo results/dovecot-append-2017-01-01-10-00-00.log
```

and have a look at the output `.svg` file in `results/`.

That's it!


## License

This project is [GPLv3](https://github.com/go-pluto/evaluation/blob/master/LICENSE) licensed.
