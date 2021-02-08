package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Motivation of this repository is to have TX Pool filled with insane numbers in geth.
// For now it will be just only a spike that makes the work, if possible it will be refactored and polished.
// It should be designed to work especially in docker and kubernetes environment, but tests at least in unit/component
// level should be runnable without containerisation.

const (
	DefaultPrivateKey    = "fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19"
	DefaultAddressToSend = "0xe86Ffce704C00556dF42e31F14CEd095390A08eF"
)

var (
	IpcEndpoint    = "./geth.ipc"
	EthereumClient *ethclient.Client
	AddressToSend  common.Address
	RoutinesCount  int
	Eth1Endpoint   string
	ChainId        *big.Int
)

type FinalReport struct {
	Errors            []error
	Transactions      []*types.Transaction
	TransactionHashes []string
}

func main() {
	var chainid int64
	flag.Int64Var(&chainid, "chain", 220720, "provide a chain id")
	flag.StringVar(&Eth1Endpoint, "endpoint", IpcEndpoint, "provide a eth1 client endpoint")
	flag.IntVar(&RoutinesCount, "routines", 1000, "provide a go routines maximum count")
	flag.Parse()
	ethClient := defaultConfig()
	fmt.Printf("\n Running chaindriller on endpoint: %s with max. routines: %d", Eth1Endpoint, RoutinesCount)
	transactionsLen := big.NewInt(int64(RoutinesCount))
	ChainId = big.NewInt(chainid)
	privateKey, err := crypto.HexToECDSA(strings.ToLower(DefaultPrivateKey))
	if nil != err {
		return
	}
	err, transactions := PrepareTransactionsForPool(transactionsLen, ethClient, privateKey)
	if nil != err {
		return
	}
	err, _ = SendBulkOfSignedTransaction(ethClient, transactions)
	if nil != err {
		return
	}
}

func PrepareTransactionsForPool(
	transactionsLen *big.Int,
	client *ethclient.Client,
	privateKey *ecdsa.PrivateKey,
) (err error, transactions []*types.Transaction) {
	ctx := context.Background()
	publicKey := privateKey.Public()
	// It will panic if public key is invalid
	publicKeyECDSA := publicKey.(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	balance, err := client.PendingBalanceAt(ctx, fromAddress)

	if nil != err {
		return
	}

	// Simple check if we have balance in this account
	if balance.Cmp(big.NewInt(0)) < 1 {
		err = fmt.Errorf("not enough balance in account address: %s", fromAddress)

		return
	}

	fmt.Printf("\n Balance of account: %d WEI", balance.Int64())

	stdInt := int(transactionsLen.Int64())

	// This is a little bit naive, but may work for the experiment if account is not used elsewhere
	nonce, err := client.PendingNonceAt(ctx, fromAddress)

	if nil != err {
		return
	}

	// lets make a tiny amount to send to not burn everything at once
	amount := big.NewInt(1)

	gasPrice, err := client.SuggestGasPrice(ctx)

	if nil != err {
		return
	}

	dummyToken := make([]byte, 16)
	rand.Read(dummyToken)

	// Call gas limit only once
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From:     fromAddress,
		To:       &AddressToSend,
		Gas:      uint64(0),
		GasPrice: gasPrice,
		Value:    amount,
		Data:     dummyToken,
	})

	if nil != err {
		return
	}

	//This is very static, should be changed to above
	gasLimit = gasLimit * 10

	// Fill the transactions, maybe sign them and then push?
	for index := 0; index < stdInt; index++ {
		// Make random bytes to differ tx (May not work as expected)
		//token := make([]byte, 16)
		//rand.Read(token)
		addrToSend := AddressToSend
		currentTx := types.NewTransaction(nonce, addrToSend, amount, gasLimit, gasPrice, make([]byte, 0))
		signedTx, err := types.SignTx(currentTx, types.NewEIP155Signer(ChainId), privateKey)

		if index%10 == 0 {
			fmt.Printf("\n Signed new tx, %d", index)
		}

		if nil != err {
			err = fmt.Errorf("\n error occured at txId: %d of total: %d, err: %s", index, stdInt, err.Error())

			return err, transactions
		}

		transactions = append(transactions, signedTx)

		// Nonce get call is done only once before the loop, may lead to problems
		nonce++
	}

	return
}

func SendBulkOfSignedTransaction(
	client *ethclient.Client,
	transactions []*types.Transaction,
) (err error, finalReport FinalReport) {
	ctx := context.Background()
	finalReport.Transactions = transactions
	finalReport.Errors = make([]error, 0)
	finalReport.TransactionHashes = make([]string, 0)

	var (
		waitGroup         sync.WaitGroup
		routinesWaitGroup sync.WaitGroup
	)

	//Lets make some sense in possible routines at once with the lock. I suggest max 1k
	minRoutinesUp := len(transactions)

	if minRoutinesUp > RoutinesCount {
		minRoutinesUp = RoutinesCount
	}

	routinesWaitGroup.Add(minRoutinesUp)

	for index, transaction := range transactions {
		waitGroup.Add(1)

		if index%100 == 0 {
			fmt.Printf("\nStarting routine index: %d", index)
		}

		go func(transaction *types.Transaction, index int) {
			routinesWaitGroup.Done()
			routinesWaitGroup.Wait()

			if index%1000 == 0 {
				fmt.Printf("\nStarting routines : %d", index)
			}

			err = client.SendTransaction(ctx, transaction)
			transactionHash := transaction.Hash()

			if nil != err {
				finalReport.Errors = append(finalReport.Errors, err)
			}

			finalReport.TransactionHashes = append(finalReport.TransactionHashes, transactionHash.String())
			waitGroup.Done()
		}(transaction, index)
	}

	waitGroup.Wait()

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

func defaultConfig() *ethclient.Client {
	chainId := os.Getenv("CHAIN_ID")
	addressToSend := os.Getenv("ADDRESS_TO_SEND")
	privateKeySender := os.Getenv("PRIVATE_KEY_SENDER")

	if "" == privateKeySender {
		privateKeySender = DefaultPrivateKey
	}

	privateKey, err := crypto.HexToECDSA(strings.ToLower(privateKeySender))

	if nil != err {
		panic(fmt.Sprintf("Invalid private key: %s, err: %s", privateKey, err.Error()))
	}

	// Fallback to default address
	if "" == addressToSend {
		addressToSend = DefaultAddressToSend
	}

	AddressToSend = common.HexToAddress(addressToSend)

	chainIdInt, err := strconv.ParseInt(chainId, 10, 64)

	if nil == err && chainIdInt != ChainId.Int64() {
		ChainId = big.NewInt(chainIdInt)
	}

	if nil != err {
		fmt.Printf("\n %v is not a valid int, defaulting to %d err: %s \n", chainId, ChainId, err.Error())
	}

	return newClient(Eth1Endpoint)
}
