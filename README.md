# This is a chaindriller
 It is inspired by chainhammer, but is focused on filling tx pool with insane numbers via IPC/RPC

## RELAYER SUPPORT
To deploy Universal Profile use `go run ./cmd/updeployer/main.go`. 
It will deploy all needed smart contracts to L14 via staging relayer. 
L15 will be supported soon.

### Flags

* Chain ID:

`-chain` (default value is `220720`)

Example usage: `./chaindriller -chain=1`

* Endpoint/path for ethereum1 node:

`-endpoint` (default value is `./geth.ipc`)

Example usage (URL): `./chaindriller -endpoint=http://34.91.155.128:8545`

Example usage (Pat): `./chaindriller -endpoint=/home/user/geth.ipc`

* Number of maximum go routines during chaindriller execution:

`-routines` (default value is `1000`)

Example usage: `./chaindriller -routines=5000`

* Number of transactions:

`-txs` (default value is `1000`)

Example usage: `./chaindriller -txs=50000`
