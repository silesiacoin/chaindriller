package main

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/silesiacoin/chaindriller/drill"
)

// Motivation of this repository is to have TX Pool filled with insane numbers in geth.
// For now it will be just only a spike that makes the work, if possible it will be refactored and polished.
// It should be designed to work especially in docker and kubernetes environment, but tests at least in unit/component
// level should be runnable without containerisation.

const (
	defaultPrivateKey    = "fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19"
	defaultAddressToSend = "0xe86Ffce704C00556dF42e31F14CEd095390A08eF"
	defaultIPCEndpoint   = "./geth.ipc"
)

var defaultChainID = big.NewInt(1)

// Config holds configuration values required by the program
type Config struct {
	ipcEndpoint   string
	chainID       *big.Int
	addressToSend common.Address
	privateKey    *ecdsa.PrivateKey
}

var reportM = sync.Mutex{}

func main() {
	cfg := getConfig()

	ethCli, err := ethclient.Dial(cfg.ipcEndpoint)
	if err != nil {
		panic(fmt.Sprintf("Could not connect to ethereum node url: %s, Err: %s", cfg.ipcEndpoint, err.Error()))
	}

	_ = drill.New(ethCli, cfg.privateKey, cfg.addressToSend, cfg.chainID)

	fmt.Printf("\n Running chaindriller on IPC: %s", cfg.ipcEndpoint)
}

func getConfig() (cfg Config) {
	cfg.ipcEndpoint = os.Getenv("IPC_ENDPOINT")
	if cfg.ipcEndpoint == "" {
		cfg.ipcEndpoint = defaultIPCEndpoint
	}

	chainIDstr := os.Getenv("CHAIN_ID")
	chainID, err := strconv.ParseInt(chainIDstr, 10, 64)

	if err == nil {
		cfg.chainID = big.NewInt(chainID)
	} else {
		fmt.Printf("\n %v is not a valid int, defaulting to %d err: %s \n", chainIDstr, defaultChainID, err.Error())
		cfg.chainID = defaultChainID
	}

	privateKeySender := os.Getenv("PRIVATE_KEY_SENDER")
	if privateKeySender == "" {
		privateKeySender = defaultPrivateKey
	}

	privateKey, err := crypto.HexToECDSA(strings.ToLower(privateKeySender))
	if err != nil {
		panic(fmt.Sprintf("Invalid private key: %s, err: %s", privateKey, err.Error()))
	}
	cfg.privateKey = privateKey

	addressToSend := os.Getenv("ADDRESS_TO_SEND")
	if addressToSend == "" {
		addressToSend = defaultAddressToSend
	}

	cfg.addressToSend = common.HexToAddress(addressToSend)

	return
}
