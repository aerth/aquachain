#!/bin/bash

# Aquachain RPC allow IP
RPC_ALLOW_IP=${RPC_ALLOW_IP}

# Aquachain coinbase address
AQUABASE=${AQUABASE}

# Aquachain data directory (default: ~/.aquachain/
AQUACHAIN_DATADIR=${AQUACHAIN_DATADIR-/var/lib/aquachain}

# Aquachain chain (mainnet, testnet, testnet3)
AQUACHAIN_CHAIN=${AQUACHAIN_CHAIN}

# Aquachain verbosity
VERBOSITY=${VERBOSITY-3}

# Additional arguments for Aquachain
AQUACHAIN_ARGS=${AQUACHAIN_ARGS}
if [ -n "${RPC_ALLOW_IP}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --rpcallowip ${RPC_ALLOW_IP}"
fi
if [ -n "${AQUACHAIN_DATADIR}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --datadir ${AQUACHAIN_DATADIR}"
fi
if [ -n "${AQUABASE}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --aquabase ${AQUABASE}"
fi
if [ -n "${AQUACHAIN_CHAIN}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} -chain ${AQUACHAIN_CHAIN}"
fi
if [ -n "${VERBOSITY}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} -verbosity ${VERBOSITY}"
fi

# RPC allow IP
export RPC_ALLOW_IP=${RPC_ALLOW_IP}

if [ "$1" = "stop" ]; then
    echo cant stop
    exit 1
fi

echo "Starting Aquachain node with args: ${AQUACHAIN_ARGS}"

exec sh -c "/usr/local/bin/aquachain ${AQUACHAIN_ARGS}"