# aquachain devcontainer (for vscode remote containers)
# https://code.visualstudio.com/docs/devcontainers/containers

# TODO: bump base image, find more go containers at https://mcr.microsoft.com/en-us/artifact/mar/devcontainers/go/tags
FROM mcr.microsoft.com/devcontainers/go:dev-1-bookworm
RUN apt update && apt install -y \
make file ncdu tree shfmt protobuf-compiler jq

# # install everything else in /usr/local/bin/
ADD /.devcontainer/bin/install_devtools.sh /install_devtools.sh
RUN GOCACHE=off PREFIX=/usr/local GOBIN=/usr/local/bin/ /install_devtools.sh all && rm -rf /go/*

# TODO: bash.bashrc does but does /etc/profile?
RUN (echo; echo "export AQUA_DATADIR=/aquadatadir") >> /etc/profile
RUN (echo; echo "export AQUA_DATADIR=/aquadatadir") >> /etc/bash.bashrc
RUN (echo; echo 'export PATH=${PATH}:/workspaces/aquachain-dev/bin:/workspaces/aquachain/bin') >> /etc/profile
RUN (echo; echo 'export PATH=${PATH}:/workspaces/aquachain-dev/bin:/workspaces/aquachain/bin') >> /etc/bash.bashrc

# this is a mountpoint for the aquachain datadir, and is AQUA_DATADIR env var in the container
# the REMOTE_USER doesn't exist yet but the number does`
RUN mkdir -p /aquadatadir; chown ${REMOTE_USER-1000}:${REMOTE_USER-1000} /aquadatadir

# # && rm /install_devtools.sh && go clean -cache -modcache -r

# # RUN adduser --disabled-password ${REMOTE_USER} && adduser ${REMOTE_USER} sudo
# # USER vscode
# # this wont work yet because the vscode user doesnt exist yet?
# # RUN usermod -aG docker ${REMOTE_USER}


