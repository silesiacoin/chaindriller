package drill

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type FinalReport struct {
	Errors            []error
	Transactions      []*types.Transaction
	TransactionHashes []string
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
		PrivKey:      privateKey,
		AddrToSend:   addressToSend,
		ChainID:      chainID,
		Transactions: make([]*types.Transaction, 0),
	}
}

type Driller struct {
	cli          ethCli
	PrivKey      *ecdsa.PrivateKey
	AddrToSend   common.Address
	ChainID      *big.Int
	Transactions []*types.Transaction
}

func (d *Driller) PrepareTransactionsForPool(transactionsLen *big.Int) (err error) {
	ctx := context.Background()
	publicKey := d.PrivKey.Public()

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
		To:       &d.AddrToSend,
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
		addrToSend := d.AddrToSend
		currentTx := types.NewTransaction(nonce, addrToSend, amount, gasLimit, gasPrice, make([]byte, 0))
		signedTx, err := types.SignTx(currentTx, types.NewEIP155Signer(d.ChainID), d.PrivKey)

		if index%10 == 0 {
			fmt.Printf("\n Signed new tx, %d", index)
		}

		if nil != err {
			err = fmt.Errorf("\n error occured at txId: %d of total: %d, err: %s", index, stdInt, err.Error())

			return err
		}

		d.Transactions = append(d.Transactions, signedTx)

		// Nonce get call is done only once before the loop, may lead to problems
		nonce++
	}

	return
}

func (d *Driller) SendBulkOfSignedTransaction() (err error, finalReport FinalReport) {
	ctx := context.Background()
	finalReport.Transactions = d.Transactions
	finalReport.Errors = make([]error, 0)
	finalReport.TransactionHashes = make([]string, 0)

	var (
		waitGroup         sync.WaitGroup
		routinesWaitGroup sync.WaitGroup
	)

	//Lets make some sense in possible routines at once with the lock. I suggest max 1k
	minRoutinesUp := len(d.Transactions)

	if minRoutinesUp > 5000 {
		minRoutinesUp = 5000
	}

	routinesWaitGroup.Add(minRoutinesUp)

	type result struct {
		hash string
		err  error
	}

	results := make(chan result, 32)

	for index, transaction := range d.Transactions {
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

			results <- result{transactionHash.String(), err}
			waitGroup.Done()
		}(transaction, index)
	}

	wrote := make(chan int)
	go func() {
		for r := range results {
			if err != nil {
				finalReport.Errors = append(finalReport.Errors, r.err)
			}

			finalReport.TransactionHashes = append(finalReport.TransactionHashes, r.hash)
		}
		wrote <- 0
	}()

	waitGroup.Wait()
	close(results)
	<-wrote

	return
}
