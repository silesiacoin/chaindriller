package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

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
	defaultChainID       = 1
)

// Config holds configuration values required by the program
type Config struct {
	addressToSend common.Address
	chainID       int64
	ipcEndpoint   string
	privateKey    *ecdsa.PrivateKey
	routinesN     int
	txN           int64
}

func main() {
	cfg := getConfig()

	// Values with no default (and not handled in getConfig) need default value
	flag.Int64Var(&cfg.chainID, "chain", cfg.chainID, "provide a chain id")
	flag.StringVar(&cfg.ipcEndpoint, "endpoint", cfg.ipcEndpoint, "provide a eth1 client endpoint")
	flag.IntVar(&cfg.routinesN, "routines", 1000, "provide a go routines maximum count")
	flag.Int64Var(&cfg.txN, "txs", 1000, "provide a transactions count")
	flag.Parse()

	fmt.Printf("\n Running chaindriller on endpoint: %s with max. routines: %d", cfg.ipcEndpoint, cfg.routinesN)

	ethCli, err := ethclient.Dial(cfg.ipcEndpoint)
	if err != nil {
		panic(fmt.Sprintf("Could not connect to ethereum node url: %s, Err: %s", cfg.ipcEndpoint, err.Error()))
	}

	d := drill.New(ethCli, cfg.privateKey, cfg.addressToSend, big.NewInt(cfg.chainID))
	d.RoutinesN = cfg.routinesN

	err = d.PrepareTransactionsForPool(big.NewInt(cfg.txN))
	if nil != err {
		return
	}

	err, _ = d.SendBulkOfSignedTransaction()
	if nil != err {
		return
	}
}

func getConfig() (cfg Config) {
	var err error

	cfg.ipcEndpoint = os.Getenv("IPC_ENDPOINT")
	if cfg.ipcEndpoint == "" {
		cfg.ipcEndpoint = defaultIPCEndpoint
	}

	chainIDstr := os.Getenv("CHAIN_ID")
	cfg.chainID, err = strconv.ParseInt(chainIDstr, 10, 64)

	if err != nil {
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
