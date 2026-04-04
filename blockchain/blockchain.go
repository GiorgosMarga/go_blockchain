package blockchain

import (
	"crypto/ecdsa"
	"encoding/gob"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"time"

	"github.com/GiorgosMarga/blockchain/block"
	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/params"
	"github.com/GiorgosMarga/blockchain/transaction"
	"github.com/GiorgosMarga/blockchain/utils"
)

var (
	ErrInvalidPrevHash  = errors.New("invalid prev hash")
	ErrInvalidTarget    = errors.New("invalid target")
	ErrInvalidMerkle    = errors.New("invalid merkle root")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	ErrInvalidValues    = errors.New("invalid values")
)

type Blockchain struct {
	Blocks  []*block.Block
	Utxos   map[crypto.Hash]utils.UtxoEntry
	Mempool map[crypto.Hash]utils.MempoolEntry
	Target  crypto.Hash
	config  params.ChainConfig
}

func New(cfg params.ChainConfig) *Blockchain {
	return &Blockchain{
		Blocks:  make([]*block.Block, 0),
		Utxos:   make(map[crypto.Hash]utils.UtxoEntry),
		Mempool: make(map[crypto.Hash]utils.MempoolEntry),
		Target:  cfg.MinTarget,
		config:  cfg,
	}
}

func (b *Blockchain) AddBlock(block *block.Block) error {
	if len(b.Blocks) == 0 {
		// add the first block in the chain
		if block.Header.PrevBlockHash != crypto.Zero() {
			return fmt.Errorf("%w: expected: %x, got %x\n", ErrInvalidPrevHash, crypto.Zero(), block.Header.PrevBlockHash)
		}
	} else {
		// check if previous block hash matches
		lastBlock := b.Blocks[len(b.Blocks)-1]
		if lastBlock.BlockHash != block.Header.PrevBlockHash {
			return fmt.Errorf("%w: expected: %x, got %x\n", ErrInvalidPrevHash, lastBlock.BlockHash, block.Header.PrevBlockHash)
		}

		// check if block matches the target
		if !block.BlockHash.Matches(b.Target) {
			return ErrInvalidTarget
		}

		// check merkle
		if !crypto.CalculateMerkleRoot(block.Txs).Matches(block.Header.MerkleRoot) {
			return ErrInvalidMerkle
		}

		if block.Header.Timestamp <= lastBlock.Header.Timestamp {
			return ErrInvalidTimestamp
		}

		if err := block.VerifyTransactions(len(b.Blocks), b.Utxos); err != nil {
			return err
		}
	}

	// remove from mempool all txs that are in the block
	for _, tx := range block.Txs {
		delete(b.Mempool, tx.Hash())
	}
	b.Blocks = append(b.Blocks, block)
	b.TryAdjustTarget()
	return nil
}

func (b *Blockchain) TryAdjustTarget() {
	if len(b.Blocks)%int(b.config.HalvingInterval) != 0 {
		return
	}

	fmt.Println("Adjusting target...")

	startTs := b.Blocks[len(b.Blocks)-1-50].Header.Timestamp

	endTs := b.Blocks[len(b.Blocks)-1].Header.Timestamp

	timeDiff := endTs - startTs
	targetSecs := int64(b.config.IdealBlockTime) * int64(b.config.DifficultyUpdateInterval)

	targetInt := new(big.Int)
	targetInt.SetBytes(b.Target[:])

	newTargetMul := targetInt.Mul(targetInt, big.NewInt(timeDiff))
	newTarget := newTargetMul.Div(newTargetMul, big.NewInt(targetSecs))

	if newTarget.Cmp(targetInt.Div(targetInt, big.NewInt(4))) == -1 {
		newTarget = targetInt.Div(targetInt, big.NewInt(4))
	} else if newTarget.Cmp(targetInt.Mul(targetInt, big.NewInt(4))) == 1 {
		newTarget = targetInt.Mul(targetInt, big.NewInt(4))
	}
	b.Target = crypto.Hash(newTarget.Bytes())
}
func (b *Blockchain) CalcBlockReward() uint64 {
	halvings := uint64(len(b.Blocks)) / b.config.HalvingInterval
	if halvings >= 64 {
		return 0
	}
	return (b.config.InitialReward * uint64(math.Pow(10, 8))) >> halvings
}
func (b *Blockchain) AddToMempool(tx *transaction.Transaction) error {
	// check if its not a ghost tx and that it doesnt double spend
	knownInputs := make(map[crypto.Hash]struct{})
	for _, input := range tx.Vin {
		_, exists := b.Utxos[input.PrevTxOutputHash]
		if !exists {
			return ErrInvalidPrevHash
		}
		_, exists = knownInputs[input.PrevTxOutputHash]
		if exists {
			return ErrInvalidPrevHash
		}
		knownInputs[input.PrevTxOutputHash] = struct{}{}
	}

	// conflict resolution
	for _, input := range tx.Vin {
		entryUtxo := b.Utxos[input.PrevTxOutputHash]
		if !entryUtxo.IsSpent {
			continue
		}
		var (
			found bool
		)
		// conflict, evict previous tx
		// find the tx in the mempool that is using this coin
		for _, mempoolEntry := range b.Mempool {
			for _, oldInput := range mempoolEntry.Tx.Vin {
				if oldInput.PrevTxOutputHash == input.PrevTxOutputHash {
					// Found one
					// Rollback: Set all UTXOs used by the OLD transaction to false(unspent)
					for _, inputToRelease := range mempoolEntry.Tx.Vin {
						utxo := b.Utxos[inputToRelease.PrevTxOutputHash]
						utxo.IsSpent = false
						b.Utxos[inputToRelease.PrevTxOutputHash] = utxo
					}

					// Evict: Remove the old transaction from the mempool
					delete(b.Mempool, mempoolEntry.Tx.Hash())
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	// Inputs vs outputs check
	var (
		inputVal  uint64 = 0
		outputVal uint64 = 0
	)
	for _, inputTx := range tx.Vin {
		prev := b.Utxos[inputTx.PrevTxOutputHash]
		inputVal += prev.TxOutput.Value
	}
	for _, outputTx := range tx.Vout {
		outputVal += outputTx.Value
	}
	if outputVal > inputVal {
		return ErrInvalidValues
	}

	// Mark New tx's inputs as spent
	for _, input := range tx.Vin {
		utxo := b.Utxos[input.PrevTxOutputHash]
		utxo.IsSpent = true
		b.Utxos[input.PrevTxOutputHash] = utxo
	}

	// Add to Mempool
	b.Mempool[tx.Hash()] = utils.MempoolEntry{Tx: tx, TimestampMs: time.Now().UnixMilli()}

	return nil
}

func (b *Blockchain) CleanupMempool() {
	now := time.Now().UnixMilli()
	toDelete := make([]*transaction.Transaction, 0)
	for _, mempoolEntry := range b.Mempool {
		if now-mempoolEntry.TimestampMs > int64(b.config.MaxMempoolTxAge) {
			// remove entries that are too old
			toDelete = append(toDelete, mempoolEntry.Tx)
		}
	}

	for _, tx := range toDelete {
		// unmark all utxo this tx was holding
		for _, txHash := range tx.Vin {
			entry, exists := b.Utxos[txHash.PrevTxOutputHash]
			if exists {
				entry.IsSpent = false
				b.Utxos[txHash.PrevTxOutputHash] = entry
			}
		}
		// delete from mempool
		delete(b.Mempool, tx.Hash())
	}
}

func (b *Blockchain) RebuildUtxos() {
	for _, block := range b.Blocks {
		for _, tx := range block.Txs {
			for _, txInput := range tx.Vin {
				delete(b.Utxos, txInput.PrevTxOutputHash)
			}
			for _, txOutput := range tx.Vout {
				b.Utxos[txOutput.Hash()] = utils.UtxoEntry{IsSpent: false, TxOutput: txOutput}
			}
		}
	}
}
func (b *Blockchain) GetUtxos(pubKey ecdsa.PublicKey) []utils.UtxoEntry {
	pubKeyBytes, _ := pubKey.Bytes()
	utxos := make([]utils.UtxoEntry, 0)
	for _, entry := range b.Utxos {
		if entry.TxOutput.PublicKey == crypto.Hash(pubKeyBytes) {
			utxos = append(utxos, entry)
		}
	}
	return utxos
}
func (b *Blockchain) GetTxsFromMempool() []*transaction.Transaction {

	txs := make([]*transaction.Transaction, 0, b.config.BlockTxCap)
	txsCtr := 0
	for k := range b.Mempool {
		txs = append(txs, b.Mempool[k].Tx)
		txsCtr++
		if txsCtr > int(b.config.BlockTxCap) {
			break
		}
	}
	return txs
}

func (b *Blockchain) LoadToFile(filepath string) error {
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o666)
	if err != nil {
		return err
	}

	return gob.NewEncoder(f).Encode(b)
}

func (b *Blockchain) LoadFromFile(filepath string) error {
	f, err := os.OpenFile(filepath, os.O_RDONLY, 0o666)
	if err != nil {
		return err
	}
	return gob.NewDecoder(f).Decode(b)
}
