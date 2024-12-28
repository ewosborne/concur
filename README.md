# concur
## A replacement for the parts of GNU Parallel that I like.

I like what `parallel` can do.  As a network engineer, over the years I've had to do things like copy a firmware update file to a tens or low hundreds of routers, or ping a bunch of hosts to see which ones were up, and `parallel` made those jobs really easy.

You know what I don't like about `parallel`?  It's got defaults that don't work for me (one job per core when I'm blocking on network I/O is hugely inefficient). Its CLI is convoluted (yes, it's a hard problem to solve in general, but still). It's got a 120-page manual or a 20-minute training video to teach you how to use it.  It's written in Perl.  It has this weird thing where it can't do CSV right out of the box and you have to do manual CPAN stuff.  Ain't nobody got time for any of that. And worst of all, it has that weird gold-digging 'I agree under penalty of death to cite parallel in any academic work'. I get that it's reasonable to ask for credit for writing free software, but if everyone did that then the whole open source industry would drown itself in a pool of paperwork.

So I give you `concur`. I'm never going to claim it's as fully featured as `parallel` but it does the bits I need, it's a few hundred lines of go, it has sensible defaults for things which aren't compute-bound, it has easy to read json output. It can flag jobs which exit with a return code other than zero. It can run all your jobs, or it can stop when the first one teminates. It's got an easy syntax for subbing in the thing you're iterating over (each entry in a list of hosts, for example).

It doesn't do as many clever things as `parallel` but it does the subset that I want and it does them well and it does them using `go's` built-in concurrency. Thus, `concur` (which as a verb is a [synonym](https://www.thesaurus.com/browse/parallel) for parallel).

This project is very much under active development so the screen scrapes here may not match what's in the latest code but it'll be close.

# TL;DR

```concur "dig @{{1}} ocw.mit.edu" 1.1.1.1 9.9.9.9 8.8.8.8 94.140.14.14	208.67.222.222```
runs five digs in parallel and spits out some JSON about the results.  Try it!  You'll like it!

# BIG IMPORTANT NOTE
This is a work in progress written by a guy who doesn't write code for a living. I believe this is pretty solid at its core and does what it says on the tin but it might have weird corner cases.  It is by no means idiomatic go, although I tried. PRs and comments welcome. It doesn't have many tests and could use a restructure. Maybe someday.

# building
You may want to build from scratch. I use [just](https://just.systems/) to manage building and testing so everything is in a `justfile` and done with `goreleaser` so it gets a little complicated. [Check it out](justfile).

You can do that too, or you can just run`go build`.

If you prefer, you can grab a pre-built binary from the [latest release](https://github.com/ewosborne/concur/releases/tag/v0.4.1).

# usage

````
Run commands concurrently

Usage:
  concur <command string> <list of arguments> [flags]

Flags:
      --any                 Return any (the first) command with exit code of zero
  -c, --concurrent string   Number of concurrent processes (0 = no limit), 'cpu' = one job per cpu core (default "128")
      --first               First command regardless of exit code
      --flag-errors         Print a message to stderr for all executed commands with an exit code other than zero
  -h, --help                help for concur
  -l, --log level string    Enable debug mode (one of d, i, w, e)
  -p, --pbar                Display a progress bar which ticks up once per completed job
  -t, --timeout int         Timeout in sec (0 for no timeout) (default 90)
      --token string        Token to match for replacement (default "{{1}}")
  -v, --version             version for concur
````

Concur takes some mandatory arguments. The first is the command you wish to run concurrently.  All subsequent arguments are whatever it is you want to change about what you run in parallel. It queues up all jobs and runs them on a number of worker goroutines until all jobs have finished - you control this number with `-c/--concurrent`. There is no timeout but you can add one with `--timeout`. There are also a couple of options to return before all jobs are done - `--any` and `--first`, see below.
And `--pbar` displays a progress bar which counts off as jobs finish. I find this useful for longer running jobs like `scp`ing a file to a bunch of hosts.

## Example
Here's an example which pings three different hosts:

```
concur "ping -c 1 {{1}}" www.mit.edu www.ucla.edu www.slashdot.org
```

This runs `ping -c 1` for each of those three web sites, replacing `{{1}}` with each in turn. It returns JSON to stdout, suitable for piping into `jq` or `gron`.  It sometimes complains but all complains go to stderr so even if there are errors you still get clean JSON on stdout.

There are two top-level keys in that JSON, `command` and `info`.  `info` doesn't have much in it now but `SystemRuntime` tells you how long it took to finish everything. 

`command` is where most of the fun is. Take a look at the example below. It's a list of information about each command which was run, sorted by runtime, fastest first.  This means that concur "ping -c 1 {{1}}" www.mit.edu www.ucla.edu www.slashdot.org | jq '.command[0] | '.arg' + " " + .runtime'always gives you the host which responded first, but -- spoiler alert!  -- there's a flag for that.  `concur "ping -c 1 {{1}}" www.mit.edu www.ucla.edu www.slashdot.org --any` gives you the same thing, albeit the full JSON output, not just the runtime and argument name.

```
concur "ping -c 1 {{1}}" www.mit.edu www.ucla.edu www.slashdot.org | jq '.command[0]
```

Some things to note are: `stdout` is presented as an array, with each line of stdout a separate array element. `stderr` is stored the same way. The return code from the application is in `returncode` and the runtime for the command is `runtime`. There's also a `jobstatus` code, one of

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


Here's the full JSON output from that sample ping.

```
{
 "command": [
  {
   "id": 0,
   "jobstatus": "Finished",
   "original": "ping -c 1 {{1}}",
   "substituted": "ping -c 1 www.mit.edu",
   "arg": "www.mit.edu",
   "stdout": [
    "PING e9566.dscb.akamaiedge.net (23.57.130.30): 56 data bytes",
    "64 bytes from 23.57.130.30: icmp_seq=0 ttl=59 time=7.901 ms",
    "",
    "--- e9566.dscb.akamaiedge.net ping statistics ---",
    "1 packets transmitted, 1 packets received, 0.0% packet loss",
    "round-trip min/avg/max/stddev = 7.901/7.901/7.901/0.000 ms",
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-27T16:34:50.917211-05:00",
   "endtime": "2024-12-27T16:34:50.937883-05:00",
   "runtime": "20.671667ms",
   "returncode": 0
  },
  {
   "id": 2,
   "jobstatus": "Finished",
   "original": "ping -c 1 {{1}}",
   "substituted": "ping -c 1 www.slashdot.org",
   "arg": "www.slashdot.org",
   "stdout": [
    "PING www.slashdot.org.cdn.cloudflare.net (104.18.4.215): 56 data bytes",
    "64 bytes from 104.18.4.215: icmp_seq=0 ttl=60 time=8.277 ms",
    "",
    "--- www.slashdot.org.cdn.cloudflare.net ping statistics ---",
    "1 packets transmitted, 1 packets received, 0.0% packet loss",
    "round-trip min/avg/max/stddev = 8.277/8.277/8.277/0.000 ms",
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-27T16:34:50.917172-05:00",
   "endtime": "2024-12-27T16:34:50.937845-05:00",
   "runtime": "20.672417ms",
   "returncode": 0
  },
  {
   "id": 1,
   "jobstatus": "Finished",
   "original": "ping -c 1 {{1}}",
   "substituted": "ping -c 1 www.ucla.edu",
   "arg": "www.ucla.edu",
   "stdout": [
    "PING d1zev4mn1zpfbc.cloudfront.net (18.161.21.115): 56 data bytes",
    "64 bytes from 18.161.21.115: icmp_seq=0 ttl=250 time=8.040 ms",
    "",
    "--- d1zev4mn1zpfbc.cloudfront.net ping statistics ---",
    "1 packets transmitted, 1 packets received, 0.0% packet loss",
    "round-trip min/avg/max/stddev = 8.040/8.040/8.040/nan ms",
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-27T16:34:50.917174-05:00",
   "endtime": "2024-12-27T16:34:50.93788-05:00",
   "runtime": "20.705917ms",
   "returncode": 0
  }
 ],
 "info": {
  "CoroutineLimit": 128,
  "SystemRuntime": "20ms"
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
  -h, --help                help for concur
  -l, --log level string    Enable debug mode (one of d, i, w, e)
  -p, --pbar                Display a progress bar which ticks up once per completed job
  -t, --timeout int         Timeout in sec (0 for no timeout) (default 90)
      --token string        Token to match for replacement (default "{{1}}")
  -v, --version             version for concur
```

`--any` starts all of the commands but exits when the first one with a zero exit code returns. One thing this is useful for is checking which DNS service is fastest:

```
{
 "command": [
  {
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
    ";; -\u003e\u003eHEADER\u003c\u003c- opcode: QUERY, status: NOERROR, id: 29892",
    ";; flags: qr rd ra; QUERY: 1, ANSWER: 3, AUTHORITY: 0, ADDITIONAL: 1",
    "",
    ";; OPT PSEUDOSECTION:",
    "; EDNS: version: 0, flags:; udp: 4096",
    ";; QUESTION SECTION:",
    ";www.mit.edu.\t\t\tIN\tA",
    "",
    ";; ANSWER SECTION:",
    "www.mit.edu.\t\t10\tIN\tCNAME\twww.mit.edu.edgekey.net.",
    "www.mit.edu.edgekey.net. 10\tIN\tCNAME\te9566.dscb.akamaiedge.net.",
    "e9566.dscb.akamaiedge.net. 10\tIN\tA\t104.102.98.27",
    "",
    ";; Query time: 7 msec",
    ";; SERVER: 192.168.1.1#53(192.168.1.1)",
    ";; WHEN: Fri Dec 27 16:52:44 EST 2024",
    ";; MSG SIZE  rcvd: 129",
    "",
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-27T16:52:44.109325-05:00",
   "endtime": "2024-12-27T16:52:44.127364-05:00",
   "runtime": "18.039375ms",
   "returncode": 0
  }
 ],
 "info": {
  "CoroutineLimit": 128,
  "SystemRuntime": "18ms"
 }
}
```


`--first` is like `--any` except it returns the first process to finish, regardless of error code. Is this useful?  I don't know. 

`-c, --concurrent` is the maximum number of goroutines you can have working at once. It defaults to 128. Why? Well, you need *some* sensible default limit. `go` can run with hundreds of thousands of goroutines, those aren't the problem. But all those goroutines have to do something, and in this case they exec a process. Can you run 1e5 simulatneous `scp` commands?  Probably not. If your network bandwidth didn't all vanish you'd run out of file descriptors or sockets or some other OS resource.  So - default limit 128. You can change it if you like.  `-c 1` serializes everything and is good for testing. With a default of 128, if you give it more than 128 things to iterate over (more than 128 hosts to ping, for example) it will run the pings in batches of 128.

I run [scaleTest.sh](this) as a sanity check scale test. It runs 500 `dig`s in parallel with no concurrency limit. It works fine (about half of those servers appear to be inactive now but that's OK), so the hard limit has to be north of 500. YMMV.

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

`-p, --pbar` draws a progress bar which counts down as jobs complete. Useful for longer running jobs. It looks like what you'd expect:

```
concur "scp -O rudder.zip {{1}}:" nas wamp-rat cloud --pbar
  33% |█████████████                           | (1/3) [19s:38s]
```

(_note for the pedants:_ the `-O` flag to `scp` is because my NAS is a Synology and `scp` to it doesn't work without `-O`.  I don't know why, I don't know what that flag does, it's not well documented. But it works with it and doesn't work without it.)

There's a fixed 250ms delay after the last job runs so that you can see that the progress bar finishes.  It is not counted in the system runtime. Try `concur "sleep {{1}}" 2 3 4 --pbar` and you'll see what I mean.


`-t, --timeout` sets a timeout, after which it kills all jobs and moves on with its life. The default is no timeout at all, so if you have a process which hangs, it'll just sit there indefinitely. The error message isn't elegant but it tries to give you as much data as it can get:

```
concur "sleep {{1}}" 1 2 3 4 5 -t 2
context popped, 0 jobs done{
 "command": [
  {
   "id": 0,
   "jobstatus": "Finished",
   "original": "sleep {{1}}",
   "substituted": "sleep 1",
   "arg": "1",
   "stdout": [
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-27T16:58:03.513929-05:00",
   "endtime": "2024-12-27T16:58:04.523784-05:00",
   "runtime": "1.009861334s",
   "returncode": 0
  }
 ],
 "info": {
  "CoroutineLimit": 128,
  "SystemRuntime": "2.001s"
 }
}
```

The `context popped` error comes out on stderr, so you can still pipe `concur` to `jq`.
Note that timeouts aren't perfect, they'll always be at least the value you specify plus a few milliseconds for `go` to catch up with paperwork and stuff. This means that sometimes a `sleep 2` with timeout of 2 will succeed and sometimes it'll fail.

`--token` is the token I look for in the command string to tell me where to sub in a command paremeter. The default is the literal string `{{1}}`. This just a simple string substitution under the hood, not some fancy template engine.  You can change it to any pattern you like, e.g. `./concur "ping -c 1 @@@" www.mit.edu www.ucla.edu www.slashdot.org --token @@@`.  You can probably do Little Bobby Tables stuff with this if you try, but why would you do that to yourself?
