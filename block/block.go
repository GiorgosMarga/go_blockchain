package block

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"math"

	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/params"
	tx "github.com/GiorgosMarga/blockchain/transaction"
	"github.com/GiorgosMarga/blockchain/utils"
)

var (
	ErrUnknownPrevOutput   = errors.New("unknown previous output hash")
	ErrDoubleSpending      = errors.New("double spending")
	ErrInvalidOutputAmount = errors.New("invalid output amount")
	ErrEmptyBlock          = errors.New("block has not txs")
	ErrInvalidCoinbaseTx   = errors.New("invalid coinbase tx")
	ErrSameOutputHash      = errors.New("same output tx hash detected")
	ErrInvalidSignature    = errors.New("invalid signature")
	ErrSpentTx             = errors.New("transaction has been spent")
)

type Block struct {
	Header    Header
	Txs       []*tx.Transaction
	BlockHash crypto.Hash
}

func (b *Block) Mine(steps int) bool {
	return b.Header.Mine(steps)
}

func (b *Block) Bytes() []byte {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(b); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func (b *Block) Hash() crypto.Hash {
	b.BlockHash = b.Header.CalculateHash()
	return b.BlockHash
}

func (b *Block) VerifyTransactions(predictedBlockHeight int, utxos map[crypto.Hash]utils.UtxoEntry) error {
	if len(b.Txs) == 0 {
		return ErrEmptyBlock
	}

	if err := b.VerifyCoinbaseTx(predictedBlockHeight, utxos); err != nil {
		return err
	}

	inputs := make(map[crypto.Hash]struct{})
	outputs := make(map[crypto.Hash]struct{})

	for idx, tx := range b.Txs {
		if idx == 0 {
			continue // skip coinbase tx
		}
		var inputAmount uint64 = 0
		var outputAmount uint64 = 0
		for _, inputTx := range tx.Vin {
			utxoEntry, exists := utxos[inputTx.PrevTxOutputHash]
			if !exists {
				return fmt.Errorf("%w: %s\n", ErrUnknownPrevOutput, inputTx.PrevTxOutputHash)
			}
			if _, exists := inputs[inputTx.PrevTxOutputHash]; exists {
				return ErrDoubleSpending
			}
			inputs[inputTx.PrevTxOutputHash] = struct{}{}

			if !inputTx.Signature.Verify(inputTx.PrevTxOutputHash, utxoEntry.TxOutput.PublicKey) {
				return ErrInvalidSignature
			}

			inputAmount += utxoEntry.TxOutput.Value
		}

		for _, outputTx := range tx.Vout {
			if _, exists := outputs[outputTx.Hash()]; exists {
				return ErrSameOutputHash
			}
			outputAmount += outputTx.Value
		}

		if outputAmount > inputAmount {
			return fmt.Errorf("%w: Input: %d\tOutput: %d\n", ErrInvalidOutputAmount, inputAmount, outputAmount)
		}

	}
	return nil
}

func (b *Block) VerifyCoinbaseTx(predictedBlockHeight int, utxos map[crypto.Hash]utils.UtxoEntry) error {
	coinbaseTx := b.Txs[0]
	if len(coinbaseTx.Vin) != 0 {
		return ErrInvalidCoinbaseTx
	}
	if len(coinbaseTx.Vout) == 0 {
		return ErrInvalidCoinbaseTx
	}

	minerFees, err := b.CalculateMinerFees(utxos)
	if err != nil {
		return err
	}

	blockReward := uint64(float64(params.MyConfig.InitialReward)*math.Pow(10, 8)) / uint64(math.Pow(2.0, float64(predictedBlockHeight/int(params.MyConfig.HalvingInterval))))

	var totalCoinbaseOutputs uint64 = 0
	for _, tx := range coinbaseTx.Vout {
		totalCoinbaseOutputs += tx.Value
	}

	if totalCoinbaseOutputs != blockReward+minerFees {
		return ErrInvalidCoinbaseTx
	}

	return nil
}

func (b *Block) CalculateMinerFees(utxos map[crypto.Hash]utils.UtxoEntry) (uint64, error) {
	var inputAmount uint64 = 0
	var outputAmount uint64 = 0

	for idx, trans := range b.Txs {
		if idx == 0 {
			continue // skip coinbase tx
		}
		for _, inputTx := range trans.Vin {
			utxoEntry, exists := utxos[inputTx.PrevTxOutputHash]
			if !exists {
				return 0, ErrUnknownPrevOutput
			}
			inputAmount += utxoEntry.TxOutput.Value
		}
		for _, output := range trans.Vout {
			outputAmount += output.Value
		}
	}

	return inputAmount - outputAmount, nil

}
