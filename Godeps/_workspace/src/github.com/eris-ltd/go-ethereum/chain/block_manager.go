package chain

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/chain/types"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/crypto"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/ethutil"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/event"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/logger"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/state"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/wire"
)

var statelogger = logger.NewLogger("BLOCK")

type Peer interface {
	Inbound() bool
	LastSend() time.Time
	LastPong() int64
	Host() []byte
	Port() uint16
	Version() string
	PingTime() string
	Connected() *int32
	Caps() *ethutil.Value
}

type EthManager interface {
	BlockManager() *BlockManager
	ChainManager() *ChainManager
	TxPool() *TxPool
	Broadcast(msgType wire.MsgType, data []interface{})
	PeerCount() int
	IsMining() bool
	IsListening() bool
	Peers() *list.List
	KeyManager() *crypto.KeyManager
	ClientIdentity() wire.ClientIdentity
	Db() ethutil.Database
	EventMux() *event.TypeMux
}

type BlockManager struct {
	// Mutex for locking the block processor. Blocks can only be handled one at a time
	mutex sync.Mutex
	// Canonical block chain
	bc *ChainManager
	// non-persistent key/value memory storage
	mem map[string]*big.Int
	// Proof of work used for validating
	Pow PoW
	// The ethereum manager interface
	eth EthManager
	// The managed states
	// Transiently state. The trans state isn't ever saved, validated and
	// it could be used for setting account nonces without effecting
	// the main states.
	transState *state.State
	// Mining state. The mining state is used purely and solely by the mining
	// operation.
	miningState *state.State

	// The last attempted block is mainly used for debugging purposes
	// This does not have to be a valid block and will be set during
	// 'Process' & canonical validation.
	lastAttemptedBlock *types.Block

	events event.Subscription
}

func NewBlockManager(ethereum EthManager) *BlockManager {
	sm := &BlockManager{
		mem: make(map[string]*big.Int),
		Pow: &EasyPow{},
		eth: ethereum,
		bc:  ethereum.ChainManager(),
	}
	sm.transState = ethereum.ChainManager().CurrentBlock.State().Copy()
	sm.miningState = ethereum.ChainManager().CurrentBlock.State().Copy()

	return sm
}

func (self *BlockManager) Start() {
	statelogger.Debugln("Starting block manager")
}

func (self *BlockManager) Stop() {
	statelogger.Debugln("Stopping state manager")
}

func (sm *BlockManager) CurrentState() *state.State {
	return sm.eth.ChainManager().CurrentBlock.State()
}

func (sm *BlockManager) TransState() *state.State {
	return sm.transState
}

func (sm *BlockManager) MiningState() *state.State {
	return sm.miningState
}

func (sm *BlockManager) NewMiningState() *state.State {
	sm.miningState = sm.eth.ChainManager().CurrentBlock.State().Copy()

	return sm.miningState
}

func (sm *BlockManager) ChainManager() *ChainManager {
	return sm.bc
}

func (self *BlockManager) ProcessTransactions(coinbase *state.StateObject, state *state.State, block, parent *types.Block, txs types.Transactions) (types.Receipts, types.Transactions, types.Transactions, types.Transactions, error) {
	var (
		receipts           types.Receipts
		handled, unhandled types.Transactions
		erroneous          types.Transactions
		totalUsedGas       = big.NewInt(0)
		err                error
	)

done:
	for i, tx := range txs {
		// If we are mining this block and validating we want to set the logs back to 0
		state.EmptyLogs()

		txGas := new(big.Int).Set(tx.Gas)

		cb := state.GetStateObject(coinbase.Address())
		st := NewStateTransition(cb, tx, state, block)
		err = st.TransitionState()
		if err != nil {
			statelogger.Infoln(err)
			switch {
			case IsNonceErr(err):
				err = nil // ignore error
				continue
			case IsGasLimitErr(err):
				unhandled = txs[i:]

				break done
			default:
				statelogger.Infoln(err)
				erroneous = append(erroneous, tx)
				err = nil
				continue
			}
		}

		// Update the state with pending changes
		state.Update()

		txGas.Sub(txGas, st.gas)
		cumulative := new(big.Int).Set(totalUsedGas.Add(totalUsedGas, txGas))
		receipt := types.NewReceipt(state.Root(), cumulative)
		receipt.SetLogs(state.Logs())
		receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

		// Notify all subscribers
		go self.eth.EventMux().Post(TxPostEvent{tx})

		receipts = append(receipts, receipt)
		handled = append(handled, tx)

		if ethutil.Config.Diff && ethutil.Config.DiffType == "all" {
			state.CreateOutputForDiff()
		}
	}

	block.GasUsed = totalUsedGas

	return receipts, handled, unhandled, erroneous, err
}

func (sm *BlockManager) Process(block *types.Block) (td *big.Int, msgs state.Messages, err error) {
	// Processing a blocks may never happen simultaneously
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.bc.HasBlock(block.Hash()) {
		return nil, nil, nil
	}

	if !sm.bc.HasBlock(block.PrevHash) {
		return nil, nil, ParentError(block.PrevHash)
	}
	parent := sm.bc.GetBlock(block.PrevHash)

	return sm.ProcessWithParent(block, parent)
}

func (sm *BlockManager) ProcessWithParent(block, parent *types.Block) (td *big.Int, messages state.Messages, err error) {
	sm.lastAttemptedBlock = block

	state := parent.State().Copy()

	// Defer the Undo on the Trie. If the block processing happened
	// we don't want to undo but since undo only happens on dirty
	// nodes this won't happen because Commit would have been called
	// before that.
	defer state.Reset()

	if ethutil.Config.Diff && ethutil.Config.DiffType == "all" {
		fmt.Printf("## %x %x ##\n", block.Hash(), block.Number)
	}

	receipts, err := sm.ApplyDiff(state, parent, block)
	if err != nil {
		return
	}

	txSha := types.DeriveSha(block.Transactions())
	if bytes.Compare(txSha, block.TxSha) != 0 {
		err = fmt.Errorf("validating transaction root. received=%x got=%x", block.TxSha, txSha)
		return
	}

	receiptSha := types.DeriveSha(receipts)
	if bytes.Compare(receiptSha, block.ReceiptSha) != 0 {
		err = fmt.Errorf("validating receipt root. received=%x got=%x", block.ReceiptSha, receiptSha)
		return
	}

	// Block validation
	if err = sm.ValidateBlock(block, parent); err != nil {
		statelogger.Errorln("validating block:", err)
		return
	}

	if err = sm.AccumelateRewards(state, block, parent); err != nil {
		statelogger.Errorln("accumulating reward", err)
		return
	}

	//block.receipts = receipts // although this isn't necessary it be in the future
	rbloom := types.CreateBloom(receipts)
	if bytes.Compare(rbloom, block.LogsBloom) != 0 {
		err = fmt.Errorf("unable to replicate block's bloom=%x", rbloom)
		return
	}

	state.Update()

	if !block.State().Cmp(state) {
		err = fmt.Errorf("invalid merkle root. received=%x got=%x", block.Root(), state.Root())
		return
	}

	// Calculate the new total difficulty and sync back to the db
	if td, ok := sm.CalculateTD(block); ok {
		// Sync the current block's state to the database and cancelling out the deferred Undo
		state.Sync()

		messages := state.Manifest().Messages
		state.Manifest().Reset()

		chainlogger.Infof("Processed block #%d (%x...)\n", block.Number, block.Hash()[0:4])

		sm.transState = state.Copy()

		sm.eth.TxPool().RemoveSet(block.Transactions())

		return td, messages, nil
	} else {
		return nil, nil, errors.New("total diff failed")
	}
}

func (sm *BlockManager) ApplyDiff(state *state.State, parent, block *types.Block) (receipts types.Receipts, err error) {
	coinbase := state.GetOrNewStateObject(block.Coinbase)
	coinbase.SetGasPool(block.CalcGasLimit(parent))

	// Process the transactions on to current block
	receipts, _, _, _, err = sm.ProcessTransactions(coinbase, state, block, parent, block.Transactions())
	if err != nil {
		return nil, err
	}

	return receipts, nil
}

func (sm *BlockManager) CalculateTD(block *types.Block) (*big.Int, bool) {
	uncleDiff := new(big.Int)
	for _, uncle := range block.Uncles {
		uncleDiff = uncleDiff.Add(uncleDiff, uncle.Difficulty)
	}

	// TD(genesis_block) = 0 and TD(B) = TD(B.parent) + sum(u.difficulty for u in B.uncles) + B.difficulty
	td := new(big.Int)
	td = td.Add(sm.bc.TD, uncleDiff)
	td = td.Add(td, block.Difficulty)

	// The new TD will only be accepted if the new difficulty is
	// is greater than the previous.
	if td.Cmp(sm.bc.TD) > 0 {
		return td, true

		// Set the new total difficulty back to the block chain
		//sm.bc.SetTotalDifficulty(td)
	}

	return nil, false
}

// Validates the current block. Returns an error if the block was invalid,
// an uncle or anything that isn't on the current block chain.
// Validation validates easy over difficult (dagger takes longer time = difficult)
func (sm *BlockManager) ValidateBlock(block, parent *types.Block) error {
	expd := CalcDifficulty(block, parent)
	if expd.Cmp(block.Difficulty) < 0 {
		return fmt.Errorf("Difficulty check failed for block %v, %v", block.Difficulty, expd)
	}

	diff := block.Time - parent.Time
	if diff < 0 {
		return ValidationError("Block timestamp less then prev block %v (%v - %v)", diff, block.Time, sm.bc.CurrentBlock.Time)
	}

	/* XXX
	// New blocks must be within the 15 minute range of the last block.
	if diff > int64(15*time.Minute) {
		return ValidationError("Block is too far in the future of last block (> 15 minutes)")
	}
	*/

	// Verify the nonce of the block. Return an error if it's not valid
	if !sm.Pow.Verify(block.HashNoNonce(), block.Difficulty, block.Nonce) {
		return ValidationError("Block's nonce is invalid (= %v)", ethutil.Bytes2Hex(block.Nonce))
	}

	return nil
}

func (sm *BlockManager) AccumelateRewards(state *state.State, block, parent *types.Block) error {
	reward := new(big.Int).Set(BlockReward)

	knownUncles := ethutil.Set(parent.Uncles)
	nonces := ethutil.NewSet(block.Nonce)
	for _, uncle := range block.Uncles {
		if nonces.Include(uncle.Nonce) {
			// Error not unique
			return UncleError("Uncle not unique")
		}

		uncleParent := sm.bc.GetBlock(uncle.PrevHash)
		if uncleParent == nil {
			return UncleError(fmt.Sprintf("Uncle's parent unknown (%x)", uncle.PrevHash[0:4]))
		}

		if uncleParent.Number.Cmp(new(big.Int).Sub(parent.Number, big.NewInt(6))) < 0 {
			return UncleError("Uncle too old")
		}

		if knownUncles.Include(uncle.Hash()) {
			return UncleError("Uncle in chain")
		}

		nonces.Insert(uncle.Nonce)

		r := new(big.Int)
		r.Mul(BlockReward, big.NewInt(15)).Div(r, big.NewInt(16))

		uncleAccount := state.GetAccount(uncle.Coinbase)
		uncleAccount.AddAmount(r)

		reward.Add(reward, new(big.Int).Div(BlockReward, big.NewInt(32)))
	}

	// Get the account associated with the coinbase
	account := state.GetAccount(block.Coinbase)
	// Reward amount of ether to the coinbase address
	account.AddAmount(reward)

	return nil
}

func (sm *BlockManager) GetMessages(block *types.Block) (messages []*state.Message, err error) {
	if !sm.bc.HasBlock(block.PrevHash) {
		return nil, ParentError(block.PrevHash)
	}

	sm.lastAttemptedBlock = block

	var (
		parent = sm.bc.GetBlock(block.PrevHash)
		state  = parent.State().Copy()
	)

	defer state.Reset()

	sm.ApplyDiff(state, parent, block)

	sm.AccumelateRewards(state, block, parent)

	return state.Manifest().Messages, nil
}
