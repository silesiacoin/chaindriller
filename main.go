package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-ethereum/accounts"
	"github.com/go-ethereum/core/types"
	"github.com/go-ethereum/ethclient"
	"math/big"
	"math/rand"
	"os"
	"strconv"
)

// Motivation of this repository is to have TX Pool filled with insane numbers in geth.
// For now it will be just only a spike that makes the work, if possible it will be refactored and polished.
// It should be designed to work especially in docker and kubernetes environment, but tests at least in unit/component
// level should be runnable without containerisation.

const (
	defaultAddressToSend = "0xAa923CA0a32D75f88138DcAc7096F665C94d6630"
)

var (
	IpcEndpoint    = "./geth.ipc"
	ChainId        = big.NewInt(1)
	EthereumClient *ethclient.Client
	AddressToSend  common.Address
)

func init() {
	ipcEndpoint := os.Getenv("IPC_ENDPOINT")
	chainId := os.Getenv("CHAIN_ID")
	addressToSend := os.Getenv("ADDRESS_TO_SEND")

	if "" != addressToSend {
		AddressToSend = common.HexToAddress(addressToSend)
	}

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

	EthereumClient = newClient(IpcEndpoint)
}

func main() {
	fmt.Printf("\n Running chaindriller on IPC: %s", IpcEndpoint)
}

func PrepareTransactionsForPool(
	transactionsLen *big.Int,
	account accounts.Account,
	client *ethclient.Client,
) (err error, transactions []*types.Transaction) {
	ctx := context.Background()
	currentProgress, err := client.SyncProgress(ctx)

	if nil != err {
		return
	}

	currentBlock := int64(currentProgress.CurrentBlock)
	balance, err := client.BalanceAt(ctx, account.Address, big.NewInt(currentBlock))

	if nil != err {
		return
	}

	// Simple check if we have balance in this account
	if balance.Cmp(big.NewInt(0)) < 1 {
		err = fmt.Errorf("not enough balance in account address: %s", account.Address)

		return
	}

	stdInt := int(transactionsLen.Int64())

	// This is a little bit naive, but may work for the experiment if account is not used elsewhere
	nonce, err := client.PendingNonceAt(ctx, account.Address)

	if nil != err {
		return
	}

	// lets make a tiny amount to send to not burn everything at once
	amount := big.NewInt(1)

	gasPrice, err := client.SuggestGasPrice(ctx)

	if nil != err {
		return
	}

	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From:     account.Address,
		To:       &AddressToSend,
		Gas:      uint64(0),
		GasPrice: gasPrice,
		Value:    nil,
		Data:     nil,
	})

	if nil != err {
		return
	}

	// Fill the transactions, maybe sign them and then push?
	for i := 0; i < stdInt; i++ {
		// Make random bytes to differ tx (May not work as expected)
		token := make([]byte, 4)
		rand.Read(token)
		currentTx := types.NewTransaction(nonce, AddressToSend, amount, gasLimit, gasPrice, token)
		transactions = append(transactions, currentTx)

		// Nonce call is done only once before the loop, may lead to problems
		nonce++
	}

	return
}

// newClient creates a client with specified remote URL.
func newClient(ipcEndpoint string) *ethclient.Client {
	client, err := ethclient.Dial(ipcEndpoint)
	if err != nil {
		panic(fmt.Sprintf("Could not connect to ethereum node url: %s, Err: %s", ipcEndpoint, err.Error()))
	}
	return client
}
