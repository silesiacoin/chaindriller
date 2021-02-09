package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
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

type FinalReport struct {
	Errors            []error
	Transactions      []*types.Transaction
	TransactionHashes []string
}

var reportM = sync.Mutex{}

func main() {
	cfg := getConfig()

	ethCli, err := ethclient.Dial(cfg.ipcEndpoint)
	if err != nil {
		panic(fmt.Sprintf("Could not connect to ethereum node url: %s, Err: %s", cfg.ipcEndpoint, err.Error()))
	}

	_ = New(ethCli, cfg.privateKey, cfg.addressToSend, cfg.chainID)

	fmt.Printf("\n Running chaindriller on IPC: %s", cfg.ipcEndpoint)
}

type ethCli interface {
	PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error)
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
}

func New(
	ethereumCli ethCli,
	privateKey *ecdsa.PrivateKey,
	addressToSend common.Address,
	chainID *big.Int) *Driller {
	return &Driller{
		cli:          ethereumCli,
		privKey:      privateKey,
		addrToSend:   addressToSend,
		chainID:      chainID,
		transactions: make([]*types.Transaction, 0),
	}
}

type Driller struct {
	cli          ethCli
	privKey      *ecdsa.PrivateKey
	addrToSend   common.Address
	chainID      *big.Int
	transactions []*types.Transaction
}

func (d *Driller) PrepareTransactionsForPool(transactionsLen *big.Int) (err error) {
	ctx := context.Background()
	publicKey := d.privKey.Public()

	// It will panic if public key is invalid
	publicKeyECDSA := publicKey.(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	balance, err := d.cli.PendingBalanceAt(ctx, fromAddress)

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
	nonce, err := d.cli.PendingNonceAt(ctx, fromAddress)
	if nil != err {
		return
	}

	// lets make a tiny amount to send to not burn everything at once
	amount := big.NewInt(1)

	gasPrice, err := d.cli.SuggestGasPrice(ctx)
	if nil != err {
		return
	}

	dummyToken := make([]byte, 16)
	rand.Read(dummyToken)

	// Call gas limit only once
	gasLimit, err := d.cli.EstimateGas(ctx, ethereum.CallMsg{
		From:     fromAddress,
		To:       &d.addrToSend,
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
		addrToSend := d.addrToSend
		currentTx := types.NewTransaction(nonce, addrToSend, amount, gasLimit, gasPrice, make([]byte, 0))
		signedTx, err := types.SignTx(currentTx, types.NewEIP155Signer(d.chainID), d.privKey)

		if index%10 == 0 {
			fmt.Printf("\n Signed new tx, %d", index)
		}

		if nil != err {
			err = fmt.Errorf("\n error occured at txId: %d of total: %d, err: %s", index, stdInt, err.Error())

			return err
		}

		d.transactions = append(d.transactions, signedTx)

		// Nonce get call is done only once before the loop, may lead to problems
		nonce++
	}

	return
}

func (d *Driller) SendBulkOfSignedTransaction() (err error, finalReport FinalReport) {
	ctx := context.Background()
	finalReport.Transactions = d.transactions
	finalReport.Errors = make([]error, 0)
	finalReport.TransactionHashes = make([]string, 0)

	var (
		waitGroup         sync.WaitGroup
		routinesWaitGroup sync.WaitGroup
	)

	//Lets make some sense in possible routines at once with the lock. I suggest max 1k
	minRoutinesUp := len(d.transactions)

	if minRoutinesUp > 5000 {
		minRoutinesUp = 5000
	}

	routinesWaitGroup.Add(minRoutinesUp)

	for index, transaction := range d.transactions {
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

			err = d.cli.SendTransaction(ctx, transaction)
			transactionHash := transaction.Hash()

			reportM.Lock()
			if nil != err {
				finalReport.Errors = append(finalReport.Errors, err)
			}

			finalReport.TransactionHashes = append(finalReport.TransactionHashes, transactionHash.String())
			reportM.Unlock()
			waitGroup.Done()
		}(transaction, index)
	}

	waitGroup.Wait()

	return
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
