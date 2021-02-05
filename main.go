package main

import (
	"fmt"
	"github.com/go-ethereum/ethclient"
	"math/big"
	"os"
	"strconv"
)

// Motivation of this repository is to have TX Pool filled with insane numbers in geth.
// For now it will be just only a spike that makes the work, if possible it will be refactored and polished.
// It should be designed to work especially in docker and kubernetes environment, but tests at least in unit/component
// level should be runnable without containerisation.

var (
	IpcEndpoint    = "./geth.ipc"
	ChainId        = big.NewInt(1)
	ethereumClient *ethclient.Client
)

func init() {
	ipcEndpoint := os.Getenv("IPC_ENDPOINT")
	chainId := os.Getenv("CHAIN_ID")

	if "" != ipcEndpoint {
		IpcEndpoint = ipcEndpoint
	}

	chainIdInt, err := strconv.ParseInt(chainId, 10, 64)

	if nil == err && chainIdInt != ChainId.Int64() {
		ChainId = big.NewInt(chainIdInt)
	}

	if nil != err {
		fmt.Printf("\n %v is not a valid int, defaulting to %d err: %s \n", chainId, ChainId, err.Error())
	}

	ethereumClient = newClient(IpcEndpoint)
}

func main() {
	fmt.Printf("\n Running chaindriller on IPC: %s", IpcEndpoint)
}

// newClient creates a client with specified remote URL.
func newClient(ipcEndpoint string) *ethclient.Client {
	client, err := ethclient.Dial(ipcEndpoint)
	if err != nil {
		panic(fmt.Sprintf("Could not connect to ethereum node url: %s, Err: %s", ipcEndpoint, err.Error()))
	}
	return client
}
