package main

import (
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
)

func TestPrepareTransactionsForPool(t *testing.T) {
	ipcPath, cleanup := mockEthIPC(t)
	defer cleanup()

	// Star client on a server
	client := newClient(ipcPath)

	privateKey, err := crypto.HexToECDSA(strings.ToLower(DefaultPrivateKey))
	assert.Nil(t, err)

	t.Run("Prepare 50 transactions", func(t *testing.T) {
		expectedLen := 50
		transactionsLen := big.NewInt(int64(expectedLen))
		err, transactions := PrepareTransactionsForPool(transactionsLen, client, privateKey)
		assert.Nil(t, err)
		assert.NotEmpty(t, transactions)
		assert.Len(t, transactions, expectedLen)

		t.Run("Nonce is increasing", func(t *testing.T) {
			firstNonce := transactions[0].Nonce()

			for index, transaction := range transactions {
				nonce := transaction.Nonce()
				assert.Equal(t, nonce, uint64(index+int(firstNonce)))
			}
		})
	})
}

// One weird scenario. When gas was set to 0 whole chain had stopped mining.
func TestSendPreparedTransactionsForPool(t *testing.T) {
	ipcPath, cleanup := mockEthIPC(t)
	defer cleanup()

	client := newClient(ipcPath)
	privateKey, err := crypto.HexToECDSA(strings.ToLower(DefaultPrivateKey))
	assert.Nil(t, err)

	defer func() {
		ChainId = big.NewInt(1)
		AddressToSend = common.Address{}
	}()

	AddressToSend = common.HexToAddress(DefaultAddressToSend)

	t.Run("Send 1000 transactions", func(t *testing.T) {
		expectedLen := 1000
		transactionsLen := big.NewInt(int64(expectedLen))
		ChainId = big.NewInt(220720)
		err, transactions := PrepareTransactionsForPool(transactionsLen, client, privateKey)
		assert.Nil(t, err)
		assert.NotEmpty(t, transactions)
		assert.Len(t, transactions, expectedLen)

		err, finalReport := SendBulkOfSignedTransaction(client, transactions)
		assert.Nil(t, err)
		assert.Len(t, finalReport.TransactionHashes, expectedLen)
		assert.Len(t, finalReport.Transactions, expectedLen)
		assert.Empty(t, finalReport.Errors)
	})
}

func mockEthIPC(t *testing.T) (path string, close func()) {
	ipcFile, err := ioutil.TempFile("", "geth_mock_*")
	if err != nil {
		log.Fatal(err)
	}
	ipcPath := ipcFile.Name()

	myAPI := rpcMethods{}

	rpcAPI := []rpc.API{
		{
			Namespace: "account",
			Public:    true,
			Service:   myAPI,
			Version:   "1.0",
		},
		{
			Namespace: "eth",
			Public:    true,
			Service:   myAPI,
			Version:   "1.0",
		},
	}

	l, s, err := rpc.StartIPCEndpoint(ipcPath, rpcAPI)
	assert.Nil(t, err)

	return ipcPath, func() {
		s.Stop()
		l.Close()
		os.Remove(ipcPath)
	}
}

type rpcMethods struct{}

func (r rpcMethods) GetBalance(addr common.Address, tag string) string {
	return "0x3E8"
}

func (r rpcMethods) GetTransactionCount(addr common.Address, tag string) string {
	return "0x0"
}

func (r rpcMethods) GasPrice() string {
	return "0x3b9aca00"
}

func (r rpcMethods) EstimateGas(msg map[string]interface{}) string {
	return "0x5208"
}

func (r rpcMethods) SendRawTransaction(data hexutil.Bytes) string {
	return "0xe670ec64341771606e55d6b4ca35a1a6b75ee3d5145a99d05921026d1527331"
}
