#!/bin/bash
# .devcontainer/install.sh 
# this script doesnt do anything but checks if it has been run
if [ ! -d /aquadatadir ]; then
    echo $(date +%Y-%m-%d:%H:%M:%S) - Warn: /aquadatadir doesntexist  | tee -a /aquadatadir/install.log
    mkdir -p /aquadatadir
    chown -R 1000:1000 /aquadatadir
fi
echo $(date +%Y-%m-%d:%H:%M:%S) - Installing aquadevcontainer dependencies | tee -a /aquadatadir/install.log
env | sort | tee -a /aquadatadir/install.log