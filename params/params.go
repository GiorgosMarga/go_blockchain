package params

import "github.com/GiorgosMarga/blockchain/crypto"

type ChainConfig struct {
	Name                     string
	InitialReward            uint64
	HalvingInterval          uint64
	IdealBlockTime           uint64
	MinTarget                crypto.Hash
	DifficultyUpdateInterval uint64
	MaxMempoolTxAge          uint64
	BlockTxCap               uint
}

var MyConfig = ChainConfig{
	Name:                     "main_config",
	InitialReward:            50,
	HalvingInterval:          210,
	IdealBlockTime:           10,
	MinTarget:                [32]byte{0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	DifficultyUpdateInterval: 50,
	MaxMempoolTxAge:          600,
	BlockTxCap:               20,
}
