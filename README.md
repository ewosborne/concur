# concur
## A replacement for the parts of GNU Parallel that I like.

I like what `parallel` can do.  As a network engineer, over the years I've had to do things like copy a firmware update file to a tens or low hundreds of routers, and `parallel` made it really easy.

You know what I don't like about `parallel`?  It's got defaults that don't work for me (one job per core when I'm blocking on network I/O is hugely inefficient). Its CLI is convoluted (yes, it's a hard problem to solve in general, but still), it has that weird gold-digging 'I agree under penalty of death to cite parallel in any academic work', it's got a $70 manual or a 20-minute training video to teach you how to use it.  It's written in Perl.  It has this weird thing where it can't do CSV right out of the box and you have to do manual CPAN stuff.  Ain't nobody got time for any of that.

So I give you `concur`. I'm never going to claim it's as fully featured as GNU parallel but it does the bits I need, it's less than 400 lines of go, it has sensible defaults for things which aren't compute-bound, it has easy to read json output. It can flag jobs which exit with a return code other than zero. It can run all your jobs, or it can stop when the first one teminates. It's got an easy syntax for subbing in the thing you're iterating over (each entry in a list of hosts, for example).

It doesn't do as many clever thins as `parallel` but it does the subset that I want and it does them well and it does them using `go's` built-in concurrency.

# usage

````
Run commands concurrently

Usage:
  concur <command string> <list of hosts> [flags]

Flags:
      --any                Any (first) command
  -c, --concurrent int     Number of concurrent processes (0 = no limit) (default 128)
      --flag-errors        Print a message to stderr for all executed commands with an exit code other than zero
  -h, --help               help for concur
  -l, --log level string   Enable debug mode (one of d, i, w, e)
  -t, --timeout int        Timeout in msec (0 for no timeout) (default 90000)
      --token string       Token to match for replacement (default "{{1}}")
  -v, --version            version for concur
````

Concur takes some mandatory arguments. The first is the command you wish to run concurrently.  All subsequent arguments are whatever it is you want to change about what you run in parallel.  Here's an example which pings three different hosts:

```
./concur "ping -c 1 {{1}}" www.mit.edu www.ucla.edu www.slashdot.org
```

This runs `ping -c 1` for each of those three web sites, replacing `{{1}}` with one of the sites. It returns a JSON block to stdout, suitable for piping into `jq` or `gron`.  

There are two top-level keys, `commands` and `info`.  `info` doesn't have much in it now. `commands` is where most of the fun is. Take a look at the example below. It's a list of information about each command which was run. It is in random order. 

Some things to note are: `stdout` is there as an array, with each line of stdout a separate array element. `stderr` is stored the same way too. The return code from the application is in `returncode` and the runtime for the command is `runtime`. 

```
{
 "commands": [
  {
   "original": "ping -c 1 {{1}}",
   "substituted": "ping -c 1 www.slashdot.org",
   "arg": "www.slashdot.org",
   "stdout": [
    "PING www.slashdot.org.cdn.cloudflare.net (104.18.4.215): 56 data bytes",
    "64 bytes from 104.18.4.215: icmp_seq=0 ttl=60 time=9.072 ms",
    "",
    "--- www.slashdot.org.cdn.cloudflare.net ping statistics ---",
    "1 packets transmitted, 1 packets received, 0.0% packet loss",
    "round-trip min/avg/max/stddev = 9.072/9.072/9.072/0.000 ms",
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-20T21:11:25.607424-05:00",
   "endtime": "2024-12-20T21:11:25.62301-05:00",
   "runtime": "15.586458ms",
   "returncode": 0
  },
  {
   "original": "ping -c 1 {{1}}",
   "substituted": "ping -c 1 www.mit.edu",
   "arg": "www.mit.edu",
   "stdout": [
    "PING e9566.dscb.akamaiedge.net (23.57.130.30): 56 data bytes",
    "64 bytes from 23.57.130.30: icmp_seq=0 ttl=59 time=4.985 ms",
    "",
    "--- e9566.dscb.akamaiedge.net ping statistics ---",
    "1 packets transmitted, 1 packets received, 0.0% packet loss",
    "round-trip min/avg/max/stddev = 4.985/4.985/4.985/nan ms",
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-20T21:11:25.607455-05:00",
   "endtime": "2024-12-20T21:11:25.625999-05:00",
   "runtime": "18.544208ms",
   "returncode": 0
  },
  {
   "original": "ping -c 1 {{1}}",
   "substituted": "ping -c 1 www.ucla.edu",
   "arg": "www.ucla.edu",
   "stdout": [
    "PING d1zev4mn1zpfbc.cloudfront.net (18.161.21.115): 56 data bytes",
    "64 bytes from 18.161.21.115: icmp_seq=0 ttl=250 time=8.651 ms",
    "",
    "--- d1zev4mn1zpfbc.cloudfront.net ping statistics ---",
    "1 packets transmitted, 1 packets received, 0.0% packet loss",
    "round-trip min/avg/max/stddev = 8.651/8.651/8.651/nan ms",
    ""
   ],
   "stderr": [
    ""
   ],
   "starttime": "2024-12-20T21:11:25.607509-05:00",
   "endtime": "2024-12-20T21:11:25.630265-05:00",
   "runtime": "22.756042ms",
   "returncode": 0
  }
 ],
 "info": {
  "systemRunTime": "22.888792ms"
 }
}
```




## Flags
`concur` has a number of useful flags:

```
      --any                Any (first) command
  -c, --concurrent int     Number of concurrent processes (0 = no limit) (default 128)
      --flag-errors        Print a message to stderr for all executed commands with an exit code other than zero
  -l, --log level string   Enable debug mode (one of d, i, w, e)
  -t, --timeout int        Timeout in msec (0 for no timeout) (default 90000)
      --token string       Token to match for replacement (default "{{1}}")
```

`--any` starts all of the commands but exits when the first one returns. It does not differentiate on error code << perhaps I should look for first successful? give a knob? >>, it just stops when one command finishes.  I've used this to ping a number of hosts, see which one was lowest, and then ssh into that one.

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

`-l, --log level` takes a single letter parameter, one of `[dwie]` - Debug, Warn, Infom, Error - and emits logs of that level or higher to stderr. It's very much a work in progress but right now looks like this:

```
./concur ls /tmp -l d > /dev/null
time=2024-12-20T21:21:55.693-05:00 level=DEBUG source=/Users/eric/prog/concur/infra/infra.go:150 msg="running command" arg=ls
```


`-t, --timeout` sets a timeout, after which it kills all jobs and moves on with its life. It's a work in progress - right now it's passed in milliseconds where I really want to pass in single-digit precision seconds. It defaults to 90seconds and that may not be a good value.  Set it to `0` if you want no timeout at all.

`--token` is the token I look for in the command string to tell me where to sub in a command paremeter. The default is the literal string `{{1}}`. This just a simple string substitution under the hood, not some fancy template engine.  You can change it to any pattern you like, e.g. `./concur "ping -c 1 @@@" www.mit.edu www.ucla.edu www.slashdot.org --token @@@`.  You can probably do Little Bobby Tables stuff with this if you try, but why would you do that to yourself?




