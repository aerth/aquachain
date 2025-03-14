# Aquachain RPC server (in no-keys mode)

#
# stage 1
#


# Build Aquachain in a stock Go builder container
FROM golang:1-alpine AS builder
RUN apk add --no-cache make musl-dev git tree file
ENV CGO_ENABLED=0
# build *this* branch
COPY . /aquachain
RUN tree /aquachain | egrep -v '\.go$'
RUN cd /aquachain && make && cd / && \
    mv /aquachain/bin/aquachain* /usr/local/bin/aquachain && \
    rm -rf /aquachain

#
# stage 2
#

# Pull Aquachain into a second stage deploy alpine container
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /usr/local/bin/aquachain /usr/local/bin/
# setup env
RUN mkdir -p /var/lib/aquachain
WORKDIR /var/lib/aquachain
RUN addgroup -S aqua && adduser -S aqua -G aqua -h /var/lib/aquachain
RUN chown -R aqua:aqua /var/lib/aquachain
USER aqua:aqua
ENV AQUA_DATADIR=/var/lib/aquachain
ENV NOKEYS=1
ENV NOSIGN=1
ENV COLOR=1
# expose ports
EXPOSE 8543 8544 21303/tcp 21303/udp
# docker build --pull --rm -f 'Dockerfile' -t 'aquachaindev:latest' '.' 
# docker run -it --rm -v ./tmpdatadir:/var/lib/aquachain -p 127.0.0.2:8543:8543 -p 127.0.0.2:8544:8544 --name aquachained aquachaindev:latest -now
# we dont know which allowip to use so leave that to runner ("-allowip", "*")
# runner should use 'daemon' as last argument to avoid console
ENTRYPOINT ["aquachain", "-rpc", "-ws", "-debug", "-behindproxy"]
