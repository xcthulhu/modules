package ar

import (
	"math/big"

	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/trie"
)

type Block interface {
	Trie() *trie.Trie
	Diff() *big.Int
}
