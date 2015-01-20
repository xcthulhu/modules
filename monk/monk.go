package monk

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"path"
	"strconv"
	"time"

	chains "github.com/eris-ltd/epm-go/chains"
	utils "github.com/eris-ltd/epm-go/utils"
	mutils "github.com/eris-ltd/modules/monkutils"
	epmtypes "github.com/eris-ltd/modules/types"

	"github.com/eris-ltd/new-thelonious/core"
	"github.com/eris-ltd/new-thelonious/core/types"
	"github.com/eris-ltd/new-thelonious/crypto"
	"github.com/eris-ltd/new-thelonious/doug"
	ethevent "github.com/eris-ltd/new-thelonious/event"
	//monklog "github.com/eris-ltd/new-thelonious/logger"
	monkstate "github.com/eris-ltd/new-thelonious/state"
	thelonious "github.com/eris-ltd/new-thelonious/thel"
	"github.com/eris-ltd/new-thelonious/thelutil"
	"github.com/eris-ltd/new-thelonious/xeth"
	"github.com/eris-ltd/thelonious/monklog"
)

//Logging
var logger *monklog.Logger = monklog.NewLogger("MONK")

func init() {
	utils.InitDecerverDir()
}

// implements epm.Blockchain
type MonkModule struct {
	monk          *Monk
	Config        *ChainConfig
	GenesisConfig *doug.GenesisConfig
}

// implements decerver-interfaces Blockchain
// this will get passed to Otto (javascript vm)
// as such, it does not have "administrative" methods
type Monk struct {
	config     *ChainConfig
	genConfig  *doug.GenesisConfig
	thelonious *thelonious.Thelonious
	pipe       *xeth.XEth
	keyManager *crypto.KeyManager
	//reactor    *monkreact.ReactorEngine
	eventMux *ethevent.TypeMux
	started  bool

	chans      map[string]chan epmtypes.Event
	reactchans map[string]<-chan interface{}
}

type Chan struct {
	ch      chan epmtypes.Event
	reactCh chan<- interface{}
	name    string
	event   string
	target  string
}

/*
   First, the functions to satisfy Module
*/

// Create a new MonkModule and internal Monk, with default config.
// Accepts a thelonious instance to yield a new
// interface into the same chain.
// It will not initialize the thelonious object for you though,
// so you can adjust configs before calling `Init()`
func NewMonk(th *thelonious.Thelonious) *MonkModule {
	mm := new(MonkModule)
	m := new(Monk)
	// Here we load default config and leave it to caller
	// to overwrite with config file or directly
	mm.Config = DefaultConfig
	m.config = mm.Config
	if th != nil {
		m.thelonious = th
	}
	m.started = false
	mm.monk = m
	return mm
}

// Configure the GenesisConfig struct
// If the chain already exists, use the provided genesis config
// TODO: move genconfig into db (safer than a config file)
//          but really we should reconstruct it from the genesis block
func (mod *MonkModule) ConfigureGenesis() {
	// first check if this chain already exists (and load genesis config from there)
	// (only if not working from a mem db)
	if !mod.Config.DbMem {
		if _, err := os.Stat(mod.Config.RootDir); err == nil {
			p := path.Join(mod.Config.RootDir, "genesis.json")
			if _, err = os.Stat(p); err == nil {
				mod.Config.GenesisConfig = p
			} else {
				//			exit(fmt.Errorf("Blockchain exists but missing genesis.json!"))
				utils.Copy(DefaultGenesisConfig, path.Join(mod.Config.RootDir, "genesis.json"))
			}
		}
	}

	// setup genesis config and genesis deploy handler
	if mod.GenesisConfig == nil {
		// fails if can't read json
		mod.GenesisConfig = mod.LoadGenesis(mod.Config.GenesisConfig)
	}
	if mod.GenesisConfig.Pdx != "" && !mod.GenesisConfig.NoGenDoug {
		// epm deploy through a pdx file
		mod.GenesisConfig.SetDeployer(func(block *types.Block) ([]byte, error) {
			// TODO: get full path
			return epmDeploy(block, mod.GenesisConfig.Pdx)
		})
	}
	mod.monk.genConfig = mod.GenesisConfig
}

// Initialize a monkchain
// It may or may not already have a thelonious instance
// Gives you a pipe, local keyMang, and reactor
// NewMonk must have been called first
func (mod *MonkModule) Init() error {

	m := mod.monk

	if m == nil {
		return fmt.Errorf("NewMonk has not been called")
	}

	// set epm contract path
	setEpmContractPath(m.config.ContractPath)
	// set the root
	// name > chainId > rootDir > default
	mod.setRootDir()
	logger.Infoln("Root directory ", mod.Config.RootDir)

	logger.Infoln("Loaded genesis configuration from: ", mod.Config.GenesisConfig)

	if !m.config.UseCheckpoint {
		m.config.LatestCheckpoint = ""
	}

	doug.Adversary = mod.Config.Adversary

	// if no thelonious instance
	if m.thelonious == nil {
		mod.thConfig()
		m.newThelonious()
	}

	m.pipe = xeth.New(m.thelonious)
	m.keyManager = m.thelonious.KeyManager()
	m.eventMux = m.thelonious.EventMux()

	// subscribe to the new block
	m.chans = make(map[string]chan epmtypes.Event)
	m.reactchans = make(map[string]<-chan interface{})

	return nil
}

// Start the thelonious node
func (mod *MonkModule) Start() (err error) {
	//startChan := mod.Subscribe("chainReady", "chainReady", "")

	m := mod.monk
	m.thelonious.Start(m.config.UseSeed) // peer seed
	m.started = true

	if m.config.Mining {
		StartMining(m.thelonious)
	}
	return nil
	/*
		seed := ""
		if mod.Config.UseSeed {
			seed = m.config.RemoteHost + ":" + strconv.Itoa(m.config.RemotePort)
		}
		m.thelonious.Start(mod.Config.Listen, seed)
		RegisterInterrupt(func(sig os.Signal) {
			m.thelonious.Stop()
			logger.Flush()
		})
		m.started = true

		if m.config.ServeRpc {
			StartRpc(m.thelonious, m.config.RpcHost, m.config.RpcPort)
		}

		m.Subscribe("newBlock", "newBlock", "")

		// wait for startup to finish
		// XXX: note for checkpoints this means waiting until
		//  the entire checkpointed state is loaded from peers...
		<-startChan
		mod.UnSubscribe("chainReady")
	*/

	return nil
}

func (mod *MonkModule) Shutdown() error {
	mod.monk.Stop()
	return nil
}

func (mod *MonkModule) ChainId() (string, error) {
	return mod.monk.ChainId()
}

func (mod *MonkModule) WaitForShutdown() {
	mod.monk.thelonious.WaitForShutdown()
}

// ReadConfig and WriteConfig implemented in config.go

// What module is this?
func (mod *MonkModule) Name() string {
	return "monk"
}

/*
   Wrapper so module satisfies Blockchain
*/

func (mod *MonkModule) WorldState() *epmtypes.WorldState {
	return mod.monk.WorldState()
}

func (mod *MonkModule) State() *epmtypes.State {
	return mod.monk.State()
}

func (mod *MonkModule) Storage(target string) *epmtypes.Storage {
	return mod.monk.Storage(target)
}

func (mod *MonkModule) Account(target string) *epmtypes.Account {
	return mod.monk.Account(target)
}

func (mod *MonkModule) StorageAt(target, storage string) string {
	return mod.monk.StorageAt(target, storage)
}

func (mod *MonkModule) BlockCount() int {
	return mod.monk.BlockCount()
}

func (mod *MonkModule) LatestBlock() string {
	return mod.monk.LatestBlock()
}

func (mod *MonkModule) Block(hash string) *epmtypes.Block {
	return mod.monk.Block(hash)
}

func (mod *MonkModule) IsScript(target string) bool {
	return mod.monk.IsScript(target)
}

func (mod *MonkModule) Tx(addr, amt string) (string, error) {
	return mod.monk.Tx(addr, amt)
}

func (mod *MonkModule) Msg(addr string, data []string) (string, error) {
	return mod.monk.Msg(addr, data)
}

func (mod *MonkModule) Script(code string) (string, error) {
	return mod.monk.Script(code)
}

func (mod *MonkModule) Transact(addr, value, gas, gasprice, data string) (string, error) {
	return mod.monk.Transact(addr, value, gas, gasprice, data)
}

func (mod *MonkModule) Subscribe(name, event, target string) chan epmtypes.Event {
	return mod.monk.Subscribe(name, event, target)
}

func (mod *MonkModule) UnSubscribe(name string) {
	mod.monk.UnSubscribe(name)
}

func (mod *MonkModule) Commit() {
	mod.monk.Commit()
}

func (mod *MonkModule) AutoCommit(toggle bool) {
	mod.monk.AutoCommit(toggle)
}

func (mod *MonkModule) IsAutocommit() bool {
	return mod.monk.IsAutocommit()
}

/*
   Module should also satisfy KeyManager
*/

func (mod *MonkModule) ActiveAddress() string {
	return mod.monk.ActiveAddress()
}

func (mod *MonkModule) Address(n int) (string, error) {
	return mod.monk.Address(n)
}

func (mod *MonkModule) SetAddress(addr string) error {
	return mod.monk.SetAddress(addr)
}

func (mod *MonkModule) SetAddressN(n int) error {
	return mod.monk.SetAddressN(n)
}

func (mod *MonkModule) NewAddress(set bool) string {
	return mod.monk.NewAddress(set)
}

func (mod *MonkModule) AddressCount() int {
	return mod.monk.AddressCount()
}

/*
   Module should satisfy a P2P interface
   Not in decerver-interfaces yet but prototyping here
*/

func (mod *MonkModule) Listen(should bool) {
	mod.monk.Listen(should)
}

/*
   Non-interface functions that otherwise prove useful
    in standalone applications, testing, and debuging
*/

// Load genesis json file (so calling pkg need not import doug)
func (mod *MonkModule) LoadGenesis(file string) *doug.GenesisConfig {
	g := doug.LoadGenesis(file)
	return g
}

// Set the genesis json object. This can only be done once
func (mod *MonkModule) SetGenesis(genJson *doug.GenesisConfig) {
	// reset the permission model struct (since config may have changed)
	//genJson.SetModel(doug.NewPermModel(genJson))
	mod.GenesisConfig = genJson
}

func (mod *MonkModule) MonkState() *monkstate.StateDB {
	return mod.monk.pipe.World().State()
}

/*
   Implement Blockchain
*/

func (monk *Monk) ChainId() (string, error) {
	// get the chain id
	/*data, err := thelutil.Config.Db.Get([]byte("ChainID"))
	if err != nil {
		return "", err
	} else if len(data) == 0 {
		return "", fmt.Errorf("ChainID is empty!")
	}
	chainId := thelutil.Bytes2Hex(data)
	return chainId, nil*/
	return "nilfornow", nil
}

func (monk *Monk) WorldState() *epmtypes.WorldState {
	state := monk.pipe.World().State()
	stateMap := &epmtypes.WorldState{make(map[string]*epmtypes.Account), []string{}}

	it := state.Trie().Iterator()
	for it.Next() { //(func(addr string, acct *thelutil.Value) {
		addr := it.Key
		//acct := it.Value
		hexAddr := thelutil.Bytes2Hex([]byte(addr))
		stateMap.Order = append(stateMap.Order, hexAddr)
		stateMap.Accounts[hexAddr] = monk.Account(hexAddr)

	}
	return stateMap
}

func (monk *Monk) State() *epmtypes.State {
	state := monk.pipe.World().State()
	stateMap := &epmtypes.State{make(map[string]*epmtypes.Storage), []string{}}

	it := state.Trie().Iterator()
	for it.Next() { //(func(addr string, acct *thelutil.Value) {
		addr := it.Key
		//acct := it.Value
		hexAddr := thelutil.Bytes2Hex([]byte(addr))
		stateMap.Order = append(stateMap.Order, hexAddr)
		stateMap.State[hexAddr] = monk.Storage(hexAddr)

	}
	return stateMap
}

func (monk *Monk) Storage(addr string) *epmtypes.Storage {
	w := monk.pipe.World()
	obj := w.SafeGet(thelutil.StringToByteFunc(addr, nil)).StateObject
	ret := &epmtypes.Storage{make(map[string]string), []string{}}
	obj.EachStorage(func(k string, v *thelutil.Value) {
		kk := thelutil.Bytes2Hex([]byte(k))
		v.Decode()
		vv := thelutil.Bytes2Hex(v.Bytes())
		ret.Order = append(ret.Order, kk)
		ret.Storage[kk] = vv
	})
	return ret
}

func (monk *Monk) Account(target string) *epmtypes.Account {
	w := monk.pipe.World()
	obj := w.SafeGet(thelutil.StringToByteFunc(target, nil)).StateObject

	bal := thelutil.NewValue(obj.Balance).String()
	nonce := obj.Nonce
	script := thelutil.Bytes2Hex(obj.Code)
	storage := monk.Storage(target)
	isscript := len(storage.Order) > 0 || len(script) > 0

	return &epmtypes.Account{
		Address:  target,
		Balance:  bal,
		Nonce:    strconv.Itoa(int(nonce)),
		Script:   script,
		Storage:  storage,
		IsScript: isscript,
	}
}

func (monk *Monk) StorageAt(contract_addr string, storage_addr string) string {
	var saddr *big.Int
	if thelutil.IsHex(storage_addr) {
		saddr = thelutil.BigD(thelutil.Hex2Bytes(thelutil.StripHex(storage_addr)))
	} else {
		saddr = thelutil.Big(storage_addr)
	}

	contract_addr = thelutil.StripHex(contract_addr)
	caddr := thelutil.Hex2Bytes(contract_addr)
	w := monk.pipe.World()
	ret := w.SafeGet(caddr).GetStorage(saddr)
	if ret.IsNil() {
		return ""
	}
	return thelutil.Bytes2Hex(ret.Bytes())
}

func (monk *Monk) BlockCount() int {
	return int(monk.thelonious.ChainManager().LastBlockNumber())
}

func (monk *Monk) LatestBlock() string {
	return thelutil.Bytes2Hex(monk.thelonious.ChainManager().LastBlockHash())
}

func (monk *Monk) Block(hash string) *epmtypes.Block {
	hashBytes := thelutil.Hex2Bytes(hash)
	block := monk.thelonious.ChainManager().GetBlock(hashBytes)
	return convertBlock(block)
}

func (monk *Monk) IsScript(target string) bool {
	// is contract if storage is empty and no bytecode
	obj := monk.Account(target)
	storage := obj.Storage
	if len(storage.Order) == 0 && obj.Script == "" {
		return false
	}
	return true
}

// send a tx
func (monk *Monk) Tx(addr, amt string) (string, error) {
	keys := monk.fetchKeyPair()
	addr = thelutil.StripHex(addr)
	if addr[:2] == "0x" {
		addr = addr[2:]
	}
	byte_addr := thelutil.Hex2Bytes(addr)
	// note, NewValue will not turn a string int into a big int..
	//start := time.Now()
	//hash, err := monk.pipe.Transact(keys, byte_addr, thelutil.NewValue(thelutil.Big(amt)), thelutil.NewValue(thelutil.Big("20000000000")), thelutil.NewValue(thelutil.Big("100000")), "")
	tx, err := monk.pipe.Transact(keys, byte_addr, thelutil.NewValue(thelutil.Big(amt)), thelutil.NewValue(thelutil.Big("20000000000")), thelutil.NewValue(thelutil.Big("100000")), []byte(""))
	//dif := time.Since(start)
	//fmt.Println("pipe tx took ", dif)
	if err != nil {
		return "", err
	}
	return thelutil.Bytes2Hex(tx.Hash()), nil
}

// send a message to a contract
func (monk *Monk) Msg(addr string, data []string) (string, error) {
	packed := PackTxDataArgs(data...)
	keys := monk.fetchKeyPair()
	addr = thelutil.StripHex(addr)
	byte_addr := thelutil.Hex2Bytes(addr)
	tx, err := monk.pipe.Transact(keys, byte_addr, thelutil.NewValue(thelutil.Big("350")), thelutil.NewValue(thelutil.Big("200000000000")), thelutil.NewValue(thelutil.Big("1000000")), []byte(packed))
	if err != nil {
		return "", err
	}
	return thelutil.Bytes2Hex(tx.Hash()), nil
}

func (monk *Monk) Script(script string) (string, error) {
	script = thelutil.StripHex(script)

	keys := monk.fetchKeyPair()

	// well isn't this pretty! barf
	tx, err := monk.pipe.Transact(keys, nil, thelutil.NewValue(thelutil.Big("271")), thelutil.NewValue(thelutil.Big("200000")), thelutil.NewValue(thelutil.Big("1000000")), []byte(script))
	if err != nil {
		return "", err
	}
	return thelutil.Bytes2Hex(core.AddressFromMessage(tx)), nil
}

func (monk *Monk) Transact(addr, amt, gas, gasprice, data string) (string, error) {
	keys := monk.fetchKeyPair()
	addr = thelutil.StripHex(addr)
	byte_addr := thelutil.Hex2Bytes(addr)
	tx, err := monk.pipe.Transact(keys, byte_addr, thelutil.NewValue(thelutil.Big(amt)), thelutil.NewValue(thelutil.Big(gas)), thelutil.NewValue(thelutil.Big(gasprice)), thelutil.StringToByteFunc(data, nil))
	if err != nil {
		return "", err
	}
	return thelutil.Bytes2Hex(tx.Hash()), nil
}

// returns a chanel that will fire when address is updated
func (monk *Monk) Subscribe(name, event, target string) chan epmtypes.Event {
	var eventObj interface{}
	var subscriber ethevent.Subscription
	switch event {
	case "newBlock":
		eventObj = core.NewBlockEvent{}
		subscriber = monk.eventMux.Subscribe(eventObj)
	}

	th_ch := subscriber.Chan()

	ch := make(chan epmtypes.Event)
	monk.chans[name] = ch
	monk.reactchans[name] = th_ch

	// fire up a goroutine and broadcast module specific chan on our main chan
	go func() {
		for {
			eve, more := <-th_ch
			if !more {
				break
			}
			returnEvent := epmtypes.Event{
				Event:     event,
				Target:    target,
				Source:    "monk",
				TimeStamp: time.Now(),
			}
			switch eve := eve.(type) {
			case core.NewBlockEvent:
				block := eve.Block
				returnEvent.Resource = convertBlock(block)
			case core.TxPreEvent:
			}
			// cast resource to appropriate type
			/*
				resource := eve.Resource
				} else if tx, ok := resource.(chain.Transaction); ok {
					returnEvent.Resource = convertTx(&tx)
				} else if txFail, ok := resource.(chain.TxFail); ok {
					tx := convertTx(txFail.Tx)
					tx.Error = txFail.Err.Error()
					returnEvent.Resource = tx
				} else {
					ethlogger.Errorln("Invalid event resource type", resource)
				}*/
			ch <- returnEvent
		}
	}()
	return ch
}

func (monk *Monk) UnSubscribe(name string) {
	/*if c, ok := monk.chans[name]; ok {
		monk.reactor.Unsubscribe(c.event, c.reactCh)
		close(c.reactCh)
		close(c.ch)
		delete(monk.chans, name)
	}*/
}

// Mine a block
func (m *Monk) Commit() {
	subscriber := m.eventMux.Subscribe(core.NewBlockEvent{})
	m.StartMining()
	_ = subscriber.Chan()
	subscriber.Unsubscribe()
	v := false
	for !v {
		v = m.StopMining()
	}
}

// start and stop continuous mining
func (m *Monk) AutoCommit(toggle bool) {
	if toggle {
		m.StartMining()
	} else {
		m.StopMining()
	}
}

func (m *Monk) IsAutocommit() bool {
	return m.thelonious.IsMining()
}

/*
   Blockchain interface should also satisfy KeyManager
   All values are hex encoded
*/

// Return the active address
func (monk *Monk) ActiveAddress() string {
	keypair := monk.keyManager.KeyPair()
	addr := thelutil.Bytes2Hex(keypair.Address())
	return addr
}

// Return the nth address in the ring
func (monk *Monk) Address(n int) (string, error) {
	ring := monk.keyManager.KeyRing()
	if n >= ring.Len() {
		return "", fmt.Errorf("cursor %d out of range (0..%d)", n, ring.Len())
	}
	pair := ring.GetKeyPair(n)
	addr := thelutil.Bytes2Hex(pair.Address())
	return addr, nil
}

// Set the address
func (monk *Monk) SetAddress(addr string) error {
	n := -1
	i := 0
	ring := monk.keyManager.KeyRing()
	ring.Each(func(kp *crypto.KeyPair) {
		a := thelutil.Bytes2Hex(kp.Address())
		if a == addr {
			n = i
		}
		i += 1
	})
	if n == -1 {
		return fmt.Errorf("Address %s not found in keyring", addr)
	}
	return monk.SetAddressN(n)
}

// Set the address to be the nth in the ring
func (monk *Monk) SetAddressN(n int) error {
	return monk.keyManager.SetCursor(n)
}

// Generate a new address
func (monk *Monk) NewAddress(set bool) string {
	newpair := crypto.GenerateNewKeyPair()
	addr := thelutil.Bytes2Hex(newpair.Address())
	ring := monk.keyManager.KeyRing()
	ring.AddKeyPair(newpair)
	if set {
		monk.SetAddressN(ring.Len() - 1)
	}
	return addr
}

// Return the number of available addresses
func (monk *Monk) AddressCount() int {
	return monk.keyManager.KeyRing().Len()
}

/*
   P2P interface
*/

// Start and stop listening on the port
func (monk *Monk) Listen(should bool) {
	if should {
		monk.StartListening()
	} else {
		monk.StopListening()
	}
}

/*
   Helper functions
*/

// create a new thelonious instance
// expects thConfig to already have been called!
// init db, nat/upnp, thelonious struct, reactorEngine, txPool, blockChain, stateManager
func (m *Monk) newThelonious() {
	db := mutils.NewDatabase(m.config.DbName, m.config.DbMem)

	keyManager := mutils.NewKeyManager(m.config.KeyStore, m.config.RootDir, db)
	err := keyManager.Init(m.config.KeySession, m.config.KeyCursor, false)
	if err != nil {
		log.Fatal(err)
	}
	m.keyManager = keyManager

	//checkpoint := thelutil.StringToByteFunc(m.config.LatestCheckpoint, nil)

	// create the thelonious obj
	c := new(thelonious.Config)
	m.fillConfig(c)
	th, err := thelonious.New(c, m.genConfig)
	//th, err := thelonious.New(db, clientIdentity, m.keyManager, thelonious.CapDefault, false, checkpoint, m.genConfig)

	if err != nil {
		log.Fatal("Could not start node: %s\n", err)
	}

	logger.Infoln("Created thelonious node")

	m.thelonious = th
}

func (m *Monk) fillConfig(c *thelonious.Config) {
	c.Port = strconv.Itoa(m.config.ListenPort)
	//c.Name = m.config.
	c.Version = m.config.Version
	c.Identifier = m.config.ClientIdentifier
	c.KeyStore = m.config.KeyStore
	c.DataDir = m.config.RootDir
	c.LogFile = m.config.LogFile
	c.LogLevel = m.config.LogLevel
	c.KeyRing = m.config.KeySession
	c.MaxPeers = m.config.MaxPeers
	//c.NATType =
	//c.PMPGateway
	c.Shh = false
	c.Dial = false
}

// returns hex addr of gendoug
/*
func (monk *Monk) GenDoug() string {
	return thelutil.Bytes2Hex(doug.GenDougByteAddr)
}*/

func (monk *Monk) StartMining() bool {
	return StartMining(monk.thelonious)
}

func (monk *Monk) StopMining() bool {
	return StopMining(monk.thelonious)
}

func (monk *Monk) StartListening() {
	//monk.thelonious.StartListening()
}

func (monk *Monk) StopListening() {
	//monk.thelonious.StopListening()
}

/*
   some key management stuff
*/

func (monk *Monk) fetchPriv() string {
	keypair := monk.keyManager.KeyPair()
	priv := thelutil.Bytes2Hex(keypair.PrivateKey)
	return priv
}

func (monk *Monk) fetchKeyPair() *crypto.KeyPair {
	return monk.keyManager.KeyPair()
}

// this is bad but I need it for testing
// TODO: deprecate!
func (monk *Monk) FetchPriv() string {
	return monk.fetchPriv()
}

func (mod *MonkModule) Restart() error {
	if err := mod.Shutdown(); err != nil {
		return err
	}

	mk := mod.monk
	mod = NewMonk(nil)
	mod.monk = mk
	mod.Config = mk.config

	if err := mod.Init(); err != nil {
		return err
	}
	if err := mod.Start(); err != nil {
		return err
	}

	return nil

}

func (monk *Monk) Stop() {
	if !monk.started {
		logger.Infoln("can't stop: haven't even started...")
		return
	}
	monk.StopMining()
	for n, _ := range monk.chans {
		monk.UnSubscribe(n)
	}
	monk.thelonious.Stop()
	monk = &Monk{config: monk.config}
	monklog.Reset()
	monk.started = false
}

// Set the root. If it's already set, check if the
func (mod *MonkModule) setRootDir() {
	c := mod.Config
	// if RootDir is set, we're done
	if c.RootDir != "" {
		/*
			if _, err := os.Stat(path.Join(c.RootDir, "config.json")); err == nil {
				mod.ReadConfig(path.Join(c.RootDir, "config.json"))
			}*/
		return
	}

	root, _ := chains.ResolveChainDir("thelonious", c.ChainName, c.ChainId)
	if root == "" {
		c.RootDir = DefaultRoot
	} else {
		c.RootDir = root
	}
}
