package types

import (
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/state"
	"math/big"
)

type BlockProcessor interface {
	ProcessWithParent(*Block, *Block) (*big.Int, state.Messages, error)
}
