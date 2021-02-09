package drill_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/silesiacoin/chaindriller/drill"
	"github.com/silesiacoin/chaindriller/drill/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	privateKey    = "fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19"
	addressToSend = "0xe86Ffce704C00556dF42e31F14CEd095390A08eF"
	chain         = 220720
)

var chainID = big.NewInt(chain)
var address = common.HexToAddress(addressToSend)
var privKey, _ = crypto.HexToECDSA(privateKey)

func getDrill(t *testing.T, c drill.EthCli) (d *drill.Driller) {
	d = drill.New(c, privKey, address, chainID)
	return
}

func TestPrepareTransactionsForPool(t *testing.T) {
	t.Run("Prepare 50 transactions", func(t *testing.T) {
		// Given
		cli := &mocks.EthCli{}
		cli.On("PendingBalanceAt", mock.Anything, mock.Anything).Return(big.NewInt(10), nil)
		cli.On("PendingNonceAt", mock.Anything, mock.Anything).Return(uint64(0), nil)
		cli.On("SuggestGasPrice", mock.Anything).Return(big.NewInt(1_000_000_000), nil)
		cli.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(21000), nil)

		d := getDrill(t, cli)
		txN := 50

		// When

		err := d.PrepareTransactionsForPool(big.NewInt(int64(txN)))

		// Then
		assert.Nil(t, err)
		assert.NotEmpty(t, d.Transactions)
		assert.Len(t, d.Transactions, txN)

		// And
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
	t.Run("Send 1000 transactions", func(t *testing.T) {
		// Given
		cli := &mocks.EthCli{}
		cli.On("PendingBalanceAt", mock.Anything, mock.Anything).Return(big.NewInt(10), nil)
		cli.On("PendingNonceAt", mock.Anything, mock.Anything).Return(uint64(0), nil)
		cli.On("SuggestGasPrice", mock.Anything).Return(big.NewInt(1_000_000_000), nil)
		cli.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(21000), nil)
		cli.On("SendTransaction", mock.Anything, mock.Anything).Return(nil)

		d := getDrill(t, cli)
		txN := 1000

		err := d.PrepareTransactionsForPool(big.NewInt(int64(txN)))
		assert.Nil(t, err)

		// When
		err, finalReport := d.SendBulkOfSignedTransaction()

		// Then
		assert.Nil(t, err)
		assert.Len(t, finalReport.TransactionHashes, txN)
		assert.Len(t, finalReport.Transactions, txN)
		assert.Empty(t, finalReport.Errors)
	})
}
