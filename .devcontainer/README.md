# aquachain devcontainer


## Building
By default, /workspaces/aquachain/bin is added to the PATH. This makes it easy to run it from the container terminal.

And AQUA_DATADIR is set to /workspaces/aquachain/tmpdatadir to remain persistent across container restarts to save bandwidth.

```

make clean default
aquachain -ws -rpc -now -datadir tmpdatadir
```

