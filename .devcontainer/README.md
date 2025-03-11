# aquachain devcontainer


## 
By default, ./bin is added to the PATH. This makes it easy to run it from the container terminal.

And AQUA_DATADIR is set to /aquadatadir to remain persistent across container restarts to save bandwidth.


To run, forwarding RPC/WS ports to host's localhost...

```bash
make clean default
aquachain -ws -rpc
```

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
