Lots has happened since v0.4.1.


* terminology cleanup, see README.me
* target names now accepted on stdin
* basic logging infra established,  although that's very much still a work in progress, I'm very confused by how go handles logging.
* per-job timeout as well as the preexisting global timeout
* internal restructuring but it's really just lipstick on a pig


I still have more to do but I figure the timeout, debug, and stdin work is enough to merit an 0.5.0 release.

Steps to 1.0.0:
    * tests!  tests!  I have, like, none, and my code is kinda sprawling and hard to test.
    * rewrite! so that it can be tested.
    * add debugs and errors, the ability to send them is useless if I don't use them.
    * package manager support, at least brew because that's the one I use. 
