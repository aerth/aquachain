# basic explorer on 8000
# edit this dockerfile to change endpoints (they are not standard aqua port numbers)
FROM golang:1-bookworm
RUN git clone https://github.com/aquachain/explorer /srv/explorer


# edit this, the wsUrl should be null if not used
# all chain data come from ws/rpc
# supply/fourbite/index are optional but helpful
RUN cat > /srv/explorer/endpoints.json <<EOF
{
    "name": "Aquachain Explorer",
    "title": "Aquachain Explorer",
    "supportedNetworks": [
      61717561,
      617175613
    ],
    "defaultNetwork": 61717561,
    "WsUrl": "ws://localhost:8944",
    "RpcUrl": "http://localhost:8943",
    "IndexUrl": "https://c.onical.org/idb/aqua",
    "FourbiteEndpoint": "https://c.onical.org/4byte/",
    "SupplyEndpoint": "https://c.onical.org/supply",
    "noteStorage": "the above public endpoints are optional but helpful",
    "BaseApiUrl": ""
}
EOF

RUN cat >/srv/index.html <<EOF
<!DOCTYPE html>
<html>
<head>
<meta http-equiv="refresh" content="0; url=/explorer/">
</head>
</html>
EOF
WORKDIR /srv/explorer
CMD [ "sh", "-c", "go run github.com/aerth/servest@latest -i 0.0.0.0 -p 8000 -d /srv/ -single -log" ]
