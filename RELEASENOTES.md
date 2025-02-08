Release v0.5.3. The big thing here is nix support, thanks to @dko1905.
At this point `concur` is pretty stable and I'm thinking of what it would take to get to v1.0.0.

Steps to 1.0.0:
    * better debugging and error handling
    * more package manager support, at least brew because that's the one I use. 
    * perhaps a text output option - I find myself doing `concur "ssh {{1}} uptime" hostA hostB` and it would be nice to have the answer on one line without having to comb through json. We'll see.