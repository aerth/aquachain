#!/bin/bash
# start-aquachain.sh - start script for Aquachain
#
# This file is called from the aquachain.service unit file.
#
# for testing, run as aqua user (or have permissions for /var/lib/aquachain etc)
# e.g. sudo -u aqua /usr/local/bin/start-aquachain.sh

# systemd ExecStop runs as 'aqua', but just in case...
iamroot=$(id -u)
run_aquachain_cmd(){
    if [ $iamroot -eq 0 ]; then
        sudo -u aqua /usr/local/bin/aquachain "$@"
    else
        /usr/local/bin/aquachain "$@"
    fi
}
stop_aquachain_gracefully(){
    output=$(run_aquachain_cmd -debug -verbosity 4 attach -exec 'admin.shutdown();' 2>&1)
    ec=$?
    if [ $ec -ne 0 ]; then
        echo "error: failed to stop aquachain, exit code $ec" 1>&2
        return 1
    fi
    echo $output | egrep 'no such file|connection refused' 1>&2
    if [ $? -eq 0 ]; then
        echo "warn: aquachain is not running" 1>&2
        return 0
    fi
    if [ -n "$1" ]; then # wait for shutdown
        echo "waiting for aquachain to stop..."
        sleep 1
        stop_aquachain_gracefully again
    fi
    echo "aquachain stopped"
    return 0
}
if [ "$1" = "stop" ] || [ "$1" = "restart" ]; then
    stop_aquachain_gracefully
    if [ "$1" = "stop" ]; then
        exit $?
    fi
fi
if [ "$1" = "status" ]; then
    run_aquachain_cmd attach -exec 'console.log("number of peers:", net.peerCount); console.log("chain height:", aqua.blockNumber); '  2>/dev/null
    exit $?
fi
if [ -f /etc/default/aquachain ]; then
    . /etc/default/aquachain || (echo error: failed to source /etc/default/aquachain 1>&2; exit 1)
else
    echo warn: no /etc/default/aquachain
fi
if [ -f /etc/aquachain/aquachain.conf ]; then
    . /etc/aquachain/aquachain.conf || (echo error: failed to source /etc/aquachain/aquachain.conf 1>&2; exit 1)
else
    echo warn: no /etc/aquachain/aquachain.conf 1>&2
fi
export JSONLOG
export NO_SIGN
export NO_KEYS
export COLOR
if [ -z "${RPC_ALLOW_IP}" ]; then
    RPC_ALLOW_IP="${RPCALLOWIP}"
fi
if [ -z "${RPC_ALLOW_IP}" ]; then
    RPC_ALLOW_IP="127.0.0.1/24"
fi
export RPC_ALLOW_IP

# Aquachain coinbase address
AQUABASE=${AQUABASE}
# Aquachain data directory (default: ~/.aquachain/
AQUACHAIN_DATADIR=${AQUACHAIN_DATADIR}
# Aquachain chain (mainnet, testnet, testnet3)
AQUACHAIN_CHAIN=${AQUACHAIN_CHAIN}
# Aquachain verbosity NUMBER for now
VERBOSITY=${VERBOSITY-3}
# Additional arguments for Aquachain (added below)
AQUACHAIN_ARGS=${AQUACHAIN_ARGS}

# add -rpc and -ws flags
USE_RPC=1
# add -rpchost and -wshost flags
PUBLIC_RPC_MODE=${PUBLIC_RPC_MODE-0}

if [ "${RPC_ALLOW_IP}" = "none" ]; then
    RPC_ALLOW_IP=""
    USE_RPC=0
fi
if [ "${RPC_ALLOW_IP}" = "0.0.0.0/0" ]; then
    USE_RPC=0
    PUBLIC_RPC_MODE=1
fi
if [ -n "${RPC_ALLOW_IP}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --allowip \"${RPC_ALLOW_IP}\""
fi
if [ -n "${AQUACHAIN_DATADIR}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --datadir \"${AQUACHAIN_DATADIR}\""
fi
if [ -n "${AQUABASE}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --aquabase \"${AQUABASE}\""
fi
if [ -n "${AQUACHAIN_CHAIN}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} -chain \"${AQUACHAIN_CHAIN}\""
fi
if [ -n "${VERBOSITY}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} -verbosity \"${VERBOSITY}\""
fi
# use public rpc
if [ "${PUBLIC_RPC_MODE}" = "1" ]; then
    echo "warn: Serving public HTTP and WS RPC on all interfaces" 1>&2
    export NO_KEYS=1
    export NO_SIGN=1
    echo warn: public rpc mode enabled, no keys or signing allowed 1>&2
    echo warn: public rpc mode enabled, no keys or signing allowed 1>&2
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --rpc --rpcaddr 0.0.0.0 --ws --wsaddr 0.0.0.0 -nokeys"
fi
# use default localhost rpc
if [ "${USE_RPC}" = "1" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --rpc --ws"
fi
export AQUACHAIN_ARGS=${AQUACHAIN_ARGS}
echo "Starting Aquachain node with args: ${AQUACHAIN_ARGS}" 1>&2

# lol TODO: fix this arg expansion
cmdline=$(echo exec /usr/local/bin/aquachain ${AQUACHAIN_ARGS} daemon)
# /bin/sh -c 'echo NO_SIGN=${NO_SIGN} NO_KEYS=${NO_KEYS}' # test that the export worked
exec /bin/sh -c "${cmdline}"