package drill_test

import (
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/silesiacoin/chaindriller/drill"
	"github.com/stretchr/testify/assert"
)

const (
	privateKey    = "fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19"
	addressToSend = "0xe86Ffce704C00556dF42e31F14CEd095390A08eF"
)

var defaultChainID = big.NewInt(1)

func getDrill(t *testing.T) (d *drill.Driller, cleanup func()) {
	ipcPath, cleanup := mockEthIPC(t)

	// Star client on a server
	c, err := ethclient.Dial(ipcPath)
	assert.Nil(t, err)

	pk, err := crypto.HexToECDSA(privateKey)
	assert.Nil(t, err)

	addr := common.HexToAddress(addressToSend)
	d = drill.New(c, pk, addr, defaultChainID)

	return
}

func TestPrepareTransactionsForPool(t *testing.T) {
	d, c := getDrill(t)
	defer c()

	t.Run("Prepare 50 transactions", func(t *testing.T) {
		expectedLen := 50
		transactionsLen := big.NewInt(int64(expectedLen))

		err := d.PrepareTransactionsForPool(transactionsLen)
		assert.Nil(t, err)
		assert.NotEmpty(t, d.Transactions)
		assert.Len(t, d.Transactions, expectedLen)

		t.Run("Nonce is increasing", func(t *testing.T) {
			firstNonce := d.Transactions[0].Nonce()

			for index, transaction := range d.Transactions {
				nonce := transaction.Nonce()
				assert.Equal(t, nonce, uint64(index+int(firstNonce)))
			}
		})
	})
}

// One weird scenario. When gas was set to 0 whole chain had stopped mining.
func TestSendPreparedTransactionsForPool(t *testing.T) {
	d, c := getDrill(t)
	defer c()

	expectedLen := 1000
	transactionsLen := big.NewInt(int64(expectedLen))
	d.ChainID = big.NewInt(220720)

	t.Run("Send 1000 transactions", func(t *testing.T) {
		err := d.PrepareTransactionsForPool(transactionsLen)
		assert.Nil(t, err)
		assert.NotEmpty(t, d.Transactions)
		assert.Len(t, d.Transactions, expectedLen)

		err, finalReport := d.SendBulkOfSignedTransaction()
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
