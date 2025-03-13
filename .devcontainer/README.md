# aquachain devcontainer

### info

By default, ./bin is added to the PATH. This makes it easy to run it from the container terminal.

And we set `AQUA_DATADIR=/aquadatadir` to remain persistent across container restarts to save bandwidth.

Also set is `SCHEDULE_TIMEOUT=20m` for auto graceful stop after 20 minutes because this isn't a production environment. (host might have a running node etc)

All tools are installed in the container, so you can run `make generate` etc.

### Navigating

Even if the repo is called `aquachain-dev` etc, the source is also mounted to the `/workspaces/aquachain` dir.

To build correctly, terminal and run `make`.

### Forwarding rpc ports to localhost

To run, forwarding RPC/WS ports to host's localhost...

```bash
make clean default
aquachain -ws -rpc
```

You will then be asked if you want to forward it. If not, you will need to open a new terminal inside the container and attach etc.

### Example Test run with verbose log

Sometimes logging can be useful during running tests:

note: we aren't redirecting stderr since `go test -v` already redirects all to stdout...

```bash
JSONLOG=1 go test -v ./opt/console -run SomeTest -count=1 | (prettylog; prettylog)
```

or 
```bash
TESTLOGLVL=trace go test -v ./opt/console -run SomeTest -count=1
```

### npm and node

Already installed is `nvm` so you can load it up with:

```
nvm install node
```

We dont use it in this repo, but its useful for other projects that you might be opening.