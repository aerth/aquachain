// console.startup.js
// this file is embedded and executed on boot
// it might look like web.version.node exists but its a getter
// lets call getinfo once and format the output

// friendly name for algo
function algoname(version) {
    switch (version) {
        case 1: return "Ethash";
        case 2: return "Argon2id";
        case 3: return "Argon2id-B";
        case 4: return "Argon2id-C";
        default: return "Unknown";
    }
}

// fetch the info from the node
function getinfo() {
    try {
        var instance = web3.version.node; // request
    } catch (e) {
        console.log("error getting instance: " + e);
        var instance = "unknown";
    }
    var info = {
        "instance": instance,
        "chainId": Number(aqua.chainId(), 16), // request, hex->dec 
        "gasPrice": web3.fromWei(aqua.gasPrice, 'gwei'), // request (gasPrice)
    }
    try {
        info["coinbase"] = aqua.coinbase; // request
    } catch (e) {
        if (e.message.indexOf("no-keybase mode") !== -1) {
            info["coinbase"] = "NO_KEYS";
        } else {
            console.log("getting coinbase:", e);
        info["coinbase"] = undefined;
        }
     }
     var head = aqua.getBlock('latest'); // request
     var headinfo = {
         "block": head.number,
         "timestamp": head.timestamp,
         "hash": head.hash,
         "gasLimit": head.gasLimit,
         "difficulty": head.difficulty,
         "algo": head.version,
         "algoname": algoname(head.version)
        };
    info["headinfo"] = headinfo;

    try {
        this.admin && (info["datadir"] = this.admin.datadir); // request
    } catch (e) { }

    return info;
}


function welcome() {
    var info = getinfo();
    var headinfo = info.headinfo;

    console.log("  instance: " + info.instance);
    console.log("  at block: " + headinfo.block + " (" + new Date(1000 * headinfo.timestamp) + ")");
    console.log("      head: " + headinfo.hash);
    console.log("  coinbase: " + info.coinbase);
    console.log("  gasPrice: " + info.gasPrice + " gigawei");
    console.log("  gasLimit: " + headinfo.gasLimit + " units");
    var diffstr = "";
    if (headinfo.difficulty < 1000) {
    console.log("nextsigner: " + headinfo.difficulty);
    } else {
    console.log("difficulty: " + (headinfo.difficulty / 1000000.0).toFixed(2) + " MH");
    }
    console.log("   chainId: " + info.chainId);
    console.log("      algo: " + headinfo.algo + " (" + headinfo.algoname + ")");
    if (info.datadir !== undefined) {
    console.log("   datadir: " + info.datadir);
    }
}

if (true) {
    try {
    welcome();
    } catch (e) {
        console.log("error in welcome: " + e);
    }
}

function myPeers() {
    try {
        var x = admin.peers;
        for (i = 0; i < 5; i++) {
            console.log("enode://" + x[i].id + "@" + x[i].network.remoteAddress)
        }
    } catch (e) {
        console.log("error fetching peer data, maybe not permitted? " + e);
    }
}