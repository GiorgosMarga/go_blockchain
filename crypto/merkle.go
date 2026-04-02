package crypto

import "crypto/sha256"

func hashPair(lhs, rhs Hash) Hash {
	b := make([]byte, len(lhs)*2)
	copy(b, lhs[:])
	copy(b, rhs[:])
	return sha256.Sum256(b)
}

type HashEntry interface {
	Hash() Hash
}

func CalculateMerkleRoot[T HashEntry](txs []T) Hash {

	txHashes := make([]Hash, 0, len(txs))
	for _, tx := range txs {
		txHashes = append(txHashes, tx.Hash())
	}
	currentLevel := make([]Hash, len(txHashes))
	copy(currentLevel, txHashes)

	var combined Hash
	for len(currentLevel) > 1 {
		var nextLevel []Hash

		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				combined = hashPair(currentLevel[i], currentLevel[i+1])
			} else {
				combined = hashPair(currentLevel[i], currentLevel[i])
			}
			nextLevel = append(nextLevel, combined)
		}
		currentLevel = nextLevel
	}

	return currentLevel[0]
}
