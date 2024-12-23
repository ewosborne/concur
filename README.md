# concur
## A replacement for the parts of GNU Parallel that I like.

I like what `parallel` can do.  As a network engineer, over the years I've had to do things like copy a firmware update file to a tens or low hundreds of routers, and `parallel` made it really easy.

You know what I don't like about `parallel`?  It's got defaults that don't work for me (one job per core when I'm blocking on network I/O is hugely inefficient). Its CLI is convoluted (yes, it's a hard problem to solve in general, but still). It has that weird gold-digging 'I agree under penalty of death to cite parallel in any academic work'. It's got a 120-page manual or a 20-minute training video to teach you how to use it.  It's written in Perl.  It has this weird thing where it can't do CSV right out of the box and you have to do manual CPAN stuff.  Ain't nobody got time for any of that.

So I give you `concur`. I'm never going to claim it's as fully featured as GNU parallel but it does the bits I need, it's less than 400 lines of go, it has sensible defaults for things which aren't compute-bound, it has easy to read json output. It can flag jobs which exit with a return code other than zero. It can run all your jobs, or it can stop when the first one teminates. It's got an easy syntax for subbing in the thing you're iterating over (each entry in a list of hosts, for example).

It doesn't do as many clever thins as `parallel` but it does the subset that I want and it does them well and it does them using `go's` built-in concurrency. Thus, `concur` (which as a verb is a [synonym](https://www.thesaurus.com/browse/parallel) for parallel).

This project is very much under active development so the screen scrapes here may not match what's in the latest code but it'll be close.

# usage

````
Run commands concurrently

Usage:
  concur <command string> <list of hosts> [flags]

Flags:
      --any                 Return any (the first) command with exit code of zero
  -c, --concurrent string   Number of concurrent processes (0 = no limit), 'cpu' = one job per cpu core (default "128")
      --first               First command regardless of exit code
      --flag-errors         Print a message to stderr for all executed commands with an exit code other than zero
  -h, --help                help for concur
  -l, --log level string    Enable debug mode (one of d, i, w, e)
  -t, --timeout int         Timeout in sec (0 for no timeout) (default 90)
      --token string        Token to match for replacement (default "{{1}}")
  -v, --version             version for concur
````

Concur takes some mandatory arguments. The first is the command you wish to run concurrently.  All subsequent arguments are whatever it is you want to change about what you run in parallel. It queues up all jobs and runs them on a number of worker goroutines until all jobs have finished - you control this number with `-c/--concurrent`. There is no timeout but you can add one with `--timeout`. There are also a couple of options to return before all jobs are done - `--any` and `--first`, see below.

## Example
Here's an example which pings three different hosts:

```
concur "ping -c 1 {{1}}" www.mit.edu www.ucla.edu www.slashdot.org
```

This runs `ping -c 1` for each of those three web sites, replacing `{{1}}` with each in turn. It returns a JSON block to stdout, suitable for piping into `jq` or `gron`.  

There are two top-level keys, `commands` and `info`.  `info` doesn't have much in it now, just the overall runtime in milliseconds and the number of worker goroutines. `commands` is where most of the fun is. Take a look at the example below. It's a list of information about each command which was run. Host as substituted in in random order. This is by design.

Some things to note are: `stdout` is there as an array, with each line of stdout a separate array element. `stderr` is stored the same way too. The return code from the application is in `returncode` and the runtime for the command is `runtime`. There's also a `jobstatus` code, one of

```
const (
	TBD JobStatus = iota
	Started
	Running
	Finished
	Errored
)
```

Return code is only valid if `jobstatus` is `Finished` or `Errored`.

```
{
 "commands": {
  "0": {
   "id": 0,
   "jobstatus": "Finished",
   "original": "ping -c 1 {{1}}",
   "substituted": "ping -c 1 www.mit.edu",
   "arg": "www.mit.edu",
   "stdout": [
    "PING e9566.dscb.akamaiedge.net (104.102.98.27): 56 data bytes",
    "64 bytes from 104.102.98.27: icmp_seq=0 ttl=59 time=8.473 ms",
    "",
    "--- e9566.dscb.akamaiedge.net ping statistics ---",
    "1 packets transmitted, 1 packets received, 0.0% packet loss",
    "round-trip min/avg/max/stddev = 8.473/8.473/8.473/0.000 ms",
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-23T15:53:37.467437-05:00",
   "endtime": "2024-12-23T15:53:37.483424-05:00",
   "runtime": "15.98725ms",
   "returncode": 0
  },
  "1": {
   "id": 1,
   "jobstatus": "Finished",
   "original": "ping -c 1 {{1}}",
   "substituted": "ping -c 1 www.ucla.edu",
   "arg": "www.ucla.edu",
   "stdout": [
    "PING d1zev4mn1zpfbc.cloudfront.net (18.161.21.10): 56 data bytes",
    "64 bytes from 18.161.21.10: icmp_seq=0 ttl=250 time=8.252 ms",
    "",
    "--- d1zev4mn1zpfbc.cloudfront.net ping statistics ---",
    "1 packets transmitted, 1 packets received, 0.0% packet loss",
    "round-trip min/avg/max/stddev = 8.252/8.252/8.252/nan ms",
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-23T15:53:37.467258-05:00",
   "endtime": "2024-12-23T15:53:37.483419-05:00",
   "runtime": "16.161833ms",
   "returncode": 0
  },
  "2": {
   "id": 2,
   "jobstatus": "Finished",
   "original": "ping -c 1 {{1}}",
   "substituted": "ping -c 1 www.slashdot.org",
   "arg": "www.slashdot.org",
   "stdout": [
    "PING www.slashdot.org.cdn.cloudflare.net (104.18.5.215): 56 data bytes",
    "64 bytes from 104.18.5.215: icmp_seq=0 ttl=60 time=8.479 ms",
    "",
    "--- www.slashdot.org.cdn.cloudflare.net ping statistics ---",
    "1 packets transmitted, 1 packets received, 0.0% packet loss",
    "round-trip min/avg/max/stddev = 8.479/8.479/8.479/nan ms",
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-23T15:53:37.467255-05:00",
   "endtime": "2024-12-23T15:53:37.483441-05:00",
   "runtime": "16.186042ms",
   "returncode": 0
  }
 },
 "info": {
  "CoroutineLimit": 128,
  "SystemRunTime": 16
 }
}
```




## Flags
`concur` has a number of useful flags:

```
      --any                 Return any (the first) command with exit code of zero
  -c, --concurrent string   Number of concurrent processes (0 = no limit), 'cpu' = one job per cpu core (default "128")
      --first               First command regardless of exit code
      --flag-errors         Print a message to stderr for all executed commands with an exit code other than zero
  -l, --log level string    Enable debug mode (one of d, i, w, e)
  -t, --timeout int         Timeout in sec (0 -- no timeout -- is the default)
      --token string        Token to match for replacement (default "{{1}}")
```

`--any` starts all of the commands but exits when the first one with a zero exit code returns. One thing this is useful for is checking which DNS service is fastest:

```
eric@Erics-MacBook-Air concur % ./concur "dig www.mit.edu @{{1}}" 192.168.1.1 1.1.1.1 8.8.8.8 --any
{
 "commands": {
  "0": {
   "id": 0,
   "jobstatus": "Finished",
   "original": "dig www.mit.edu @{{1}}",
   "substituted": "dig www.mit.edu @192.168.1.1",
   "arg": "192.168.1.1",
   "stdout": [
    "",
    "; \u003c\u003c\u003e\u003e DiG 9.10.6 \u003c\u003c\u003e\u003e www.mit.edu @192.168.1.1",
    ";; global options: +cmd",
    ";; Got answer:",
    ";; -\u003e\u003eHEADER\u003c\u003c- opcode: QUERY, status: NOERROR, id: 3269",
    ";; flags: qr rd ra; QUERY: 1, ANSWER: 3, AUTHORITY: 0, ADDITIONAL: 1",
    "",
    ";; OPT PSEUDOSECTION:",
    "; EDNS: version: 0, flags:; udp: 4096",
    ";; QUESTION SECTION:",
    ";www.mit.edu.\t\t\tIN\tA",
    "",
    ";; ANSWER SECTION:",
    "www.mit.edu.\t\t52\tIN\tCNAME\twww.mit.edu.edgekey.net.",
    "www.mit.edu.edgekey.net. 52\tIN\tCNAME\te9566.dscb.akamaiedge.net.",
    "e9566.dscb.akamaiedge.net. 52\tIN\tA\t104.102.98.27",
    "",
    ";; Query time: 9 msec",
    ";; SERVER: 192.168.1.1#53(192.168.1.1)",
    ";; WHEN: Mon Dec 23 15:59:18 EST 2024",
    ";; MSG SIZE  rcvd: 129",
    "",
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-23T15:59:18.174357-05:00",
   "endtime": "2024-12-23T15:59:18.196622-05:00",
   "runtime": "22.265375ms",
   "returncode": 0
  }
 },
 "info": {
  "CoroutineLimit": 128,
  "SystemRunTime": 22
 }
}
```


`--first` is like `--any` except it returns the first process to finish, regardless of error code. Is this useful?  I don't know. 

`-c, --concurrent` is the maximum number of goroutines you can have working at once. It defaults to 128. Why? Well, you need *some* sensible default limit. `go` can run with hundreds of thousands of goroutines, those aren't the problem. But all those goroutines have to do something, and in my case they spawn a shell to run a program. Can you run 1e5 simulatneous `scp` commands?  Probably not. If your network bandwidth didn't all vanish you'd run out of file descriptors or sockets or some other OS resource.  So - default limit 128. You can change it if you like.  `-c 1` serializes everything and is good for testing. 

`--flag-errors` will spit a message out to stdout for every command which returns but with a non-zero exit code.  This is useful for catching commands which ran but which weren't happy about it. Here's that ping example again but with a typo:


```
./concur "ping -c 1 {{1}}" www.mit.edu www.ucla.edu www.slashdot.ogr 
```

If we run it with `--flag-errors` and dump stdout to /dev/null to make it easy to read it shows:

```
./concur "ping -c 1 {{1}}" www.mit.edu www.ucla.edu www.slashdot.ogr --flag-errors > /dev/null
command ping -c 1 www.slashdot.ogr exited with error code 68
```
68 is apparently what `ping` uses to mean `cannot resolve host`.  

`-l, --log level` takes a single letter parameter, one of `[dwie]` - Debug, Warn, Infom, Error - and emits logs of that level or higher to stderr. It's very much a work in progress but right now looks like this:

```
./concur ls /tmp -l d > /dev/null
time=2024-12-20T21:21:55.693-05:00 level=DEBUG source=/Users/eric/prog/concur/infra/infra.go:150 msg="running command" arg=ls
```

I don't have many log messages yet. Maybe someday.


`-t, --timeout` sets a timeout, after which it kills all jobs and moves on with its life. It's a work in progress - right now it's passed in milliseconds where I really want to pass in single-digit precision seconds. It defaults to 90seconds and that may not be a good value.  Set it to `0` if you want no timeout at all.

`--token` is the token I look for in the command string to tell me where to sub in a command paremeter. The default is the literal string `{{1}}`. This just a simple string substitution under the hood, not some fancy template engine.  You can change it to any pattern you like, e.g. `./concur "ping -c 1 @@@" www.mit.edu www.ucla.edu www.slashdot.org --token @@@`.  You can probably do Little Bobby Tables stuff with this if you try, but why would you do that to yourself?




