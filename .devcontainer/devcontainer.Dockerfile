# aquachain devcontainer (for vscode remote containers)
# https://code.visualstudio.com/docs/devcontainers/containers

# TODO: bump base image, find more go containers at https://mcr.microsoft.com/en-us/artifact/mar/devcontainers/go/tags
FROM mcr.microsoft.com/devcontainers/go:dev-1-bookworm
RUN apt update && apt install -y \
make file ncdu tree shfmt protobuf-compiler jq

# copy tool installer (and list of packages)
ADD /contrib/scripts/install_devtools.sh /contrib/scripts/devtools.go.list /
# copy a completion script (might need to be updated in the container depending on branch)
ADD /contrib/completion_aquachain.bash /etc/bash_completion.d/aquachain
# add additional go tools to install here
RUN echo "github.com/rs/zerolog/cmd/prettylog@latest" >> /devtools.go.list
# install and cleanup
RUN GOCACHE=off PREFIX=/usr/local GOBIN=/usr/local/bin/ /install_devtools.sh all && rm -rf /go/*
RUN go clean -cache -modcache
RUN rm /install_devtools.sh /devtools.go.list

# test the go-tools were installed
RUN test -x /usr/local/bin/stringer

# install bashrc stuff (for all users)
# TODO: bash.bashrc does but does /etc/profile?
RUN (echo; echo "export AQUA_DATADIR=/aquadatadir") | tee -a /etc/profile /etc/bash.bashrc
RUN (echo; echo 'export PATH=${PATH}:/workspaces/aquachain-dev/bin:/workspaces/aquachain/bin') | tee -a /etc/profile /etc/bash.bashrc

# this is a mountpoint for the aquachain datadir, and is AQUA_DATADIR env var in the container
# the REMOTE_USER doesn't exist yet but the number works, and we assume the dev is 1000 (or root..) in the container
RUN mkdir -p /aquadatadir; chown ${REMOTE_USER-1000}:${REMOTE_USER-1000} /aquadatadir
