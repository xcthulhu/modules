package monk

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/kardianos/osext"
	//"github.com/eris-ltd/modules/genblock"
	"github.com/eris-ltd/epm-go/epm"
	"github.com/eris-ltd/epm-go/utils"
	//mutils "github.com/eris-ltd/modules/monkutils"
	epmtypes "github.com/eris-ltd/modules/types"

	"github.com/eris-ltd/new-thelonious/core/types"
	"github.com/eris-ltd/new-thelonious/crypto"
	thel "github.com/eris-ltd/new-thelonious/thel"
	//	monklog "github.com/eris-ltd/new-thelonious/logger"
	"github.com/eris-ltd/new-thelonious/miner"
	monkstate "github.com/eris-ltd/new-thelonious/state"
	"github.com/eris-ltd/new-thelonious/thelutil"
)

// this is basically go-etheruem/utils

// TODO: use the interupts...

//var logger = logger.NewLogger("CLI")
var interruptCallbacks = []func(os.Signal){}

// Register interrupt handlers callbacks
func RegisterInterrupt(cb func(os.Signal)) {
	interruptCallbacks = append(interruptCallbacks, cb)
}

// go routine that call interrupt handlers in order of registering
func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	go func() {
		signal.Notify(c, os.Interrupt)
		for sig := range c {
			logger.Errorf("Shutting down (%v) ... \n", sig)
			RunInterruptCallbacks(sig)
		}
	}()
}

func RunInterruptCallbacks(sig os.Signal) {
	for _, cb := range interruptCallbacks {
		cb(sig)
	}
}

func confirm(message string) bool {
	fmt.Println(message, "Are you sure? (y/n)")
	var r string
	fmt.Scanln(&r)
	for ; ; fmt.Scanln(&r) {
		if r == "n" || r == "y" {
			break
		} else {
			fmt.Printf("Yes or no?", r)
		}
	}
	return r == "y"
}

// TODO: dwell on this more too
func InitConfig(ConfigFile string, Datadir string, EnvPrefix string) *thelutil.ConfigManager {
	utils.InitDataDir(Datadir)
	return thelutil.ReadConfig(ConfigFile, Datadir, EnvPrefix)
}

func exit(err error) {
	status := 0
	if err != nil {
		fmt.Println(err)
		logger.Errorln("Fatal: ", err)
		status = 1
	}
	//logger.Flush()
	os.Exit(status)
}

func ShowGenesis(ethereum *thel.Thelonious) {
	logger.Infoln(ethereum.ChainManager().Genesis())
	exit(nil)
}

// TODO: work this baby
func DefaultAssetPath() string {
	var assetPath string
	// If the current working directory is the go-ethereum dir
	// assume a debug build and use the source directory as
	// asset directory.
	pwd, _ := os.Getwd()
	if pwd == path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "ethereal") {
		assetPath = path.Join(pwd, "assets")
	} else {
		switch runtime.GOOS {
		case "darwin":
			// Get Binary Directory
			exedir, _ := osext.ExecutableFolder()
			assetPath = filepath.Join(exedir, "../Resources")
		case "linux":
			assetPath = "/usr/share/ethereal"
		case "windows":
			assetPath = "./assets"
		default:
			assetPath = "."
		}
	}
	return assetPath
}

// TODO: use this...
func KeyTasks(keyManager *crypto.KeyManager, KeyRing string, GenAddr bool, SecretFile string, ExportDir string, NonInteractive bool) {

	var err error
	switch {
	case GenAddr:
		if NonInteractive || confirm("This action overwrites your old private key.") {
			err = keyManager.Init(KeyRing, 0, true)
		}
		exit(err)
	case len(SecretFile) > 0:
		SecretFile = thelutil.ExpandHomePath(SecretFile)

		if NonInteractive || confirm("This action overwrites your old private key.") {
			err = keyManager.InitFromSecretsFile(KeyRing, 0, SecretFile)
		}
		exit(err)
	case len(ExportDir) > 0:
		err = keyManager.Init(KeyRing, 0, false)
		if err == nil {
			err = keyManager.Export(ExportDir)
		}
		exit(err)
	default:
		// Creates a keypair if none exists
		err = keyManager.Init(KeyRing, 0, false)
		if err != nil {
			exit(err)
		}
	}
}

/*func StartRpc(ethereum *thel.Thelonious, RpcHost string, RpcPort int) {
	var err error
	rpcAddr := RpcHost + ":" + strconv.Itoa(RpcPort)
	ethereum.RpcServer, err = rpc.NewJsonRpcServer(xeth.NewJSPipe(ethereum), rpcAddr)
	if err != nil {
		logger.Errorf("Could not start RPC interface (port %v): %v", RpcPort, err)
	} else {
		go ethereum.RpcServer.Start()
	}
}*/

var myminer *miner.Miner

func GetMiner() *miner.Miner {
	return myminer
}

func StartMining(ethereum *thel.Thelonious) bool {

	if !ethereum.Mining {
		ethereum.Mining = true
		addr := ethereum.KeyManager().Address()

		go func() {
			logger.Infoln("Start mining")
			if myminer == nil {
				myminer = miner.New(addr, ethereum)
			}
			// Give it some time to connect with peers
			time.Sleep(3 * time.Second)
			/*for !ethereum.IsUpToDate() {
				time.Sleep(5 * time.Second)
			}*/
			myminer.Start()
		}()
		RegisterInterrupt(func(os.Signal) {
			StopMining(ethereum)
		})
		return true
	}
	return false
}

func FormatTransactionData(data string) []byte {
	d := thelutil.StringToByteFunc(data, func(s string) (ret []byte) {
		slice := regexp.MustCompile("\\n|\\s").Split(s, 1000000000)
		for _, dataItem := range slice {
			d := thelutil.FormatData(dataItem)
			ret = append(ret, d...)
		}
		return
	})

	return d
}

func StopMining(ethereum *thel.Thelonious) bool {
	if ethereum.Mining && myminer != nil {
		myminer.Stop()
		logger.Infoln("Stopped mining")
		ethereum.Mining = false
		myminer = nil
		return true
	}

	return false
}

// Set the EPM contract root
func setEpmContractPath(p string) {
	epm.ContractPath = p
}

// Deploy a pdx onto a block
// This is used as a doug deploy function
func epmDeploy(block *types.Block, pkgDef string) ([]byte, error) {
	// TODO: use epm here
	/*
		m := genblock.NewGenBlockModule(block)
		m.Config.LogLevel = 5
		err := m.Init()
		if err != nil {
			return nil, err
		}
		m.Start()
		epm.ErrMode = epm.ReturnOnErr
		e, err := epm.NewEPM(m, ".epm-log")
		if err != nil {
			return nil, err
		}
		err = e.Parse(pkgDef)
		if err != nil {
			return nil, err
		}
		err = e.ExecuteJobs()
		if err != nil {
			return nil, err
		}
		e.Commit()
		chainId, err := m.ChainId()
		if err != nil {
			return nil, err
		}
		return chainId, nil
	*/
	return nil, nil
}

func splitHostPort(peerServer string) (string, int, error) {
	spl := strings.Split(peerServer, ":")
	if len(spl) < 2 {
		return "", 0, fmt.Errorf("Impromerly formatted peer server. Should be <host>:<port>")
	}
	host := spl[0]
	port, err := strconv.Atoi(spl[1])
	if err != nil {
		return "", 0, fmt.Errorf("Bad port number: ", spl[1])
	}
	return host, port, nil
}

/*func ChainIdFromDb(root string) (string, error) {
	thelutil.Config = &thelutil.ConfigManager{ExecPath: root, Debug: true, Paranoia: true}
	db := mutils.NewDatabase("database", false)
	thelutil.Config.Db = db
	data, err := thelutil.Config.Db.Get([]byte("ChainID"))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("Empty ChainID!")
	}
	return thelutil.Bytes2Hex(data), nil
}*/

func rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func copy(oldpath, newpath string) error {
	return utils.Copy(oldpath, newpath)
}

// some convenience functions

// get users home directory
func homeDir() string {
	usr, _ := user.Current()
	return usr.HomeDir
}

// convert a big int from string to hex
func BigNumStrToHex(s string) string {
	bignum := thelutil.Big(s)
	bignum_bytes := thelutil.BigToBytes(bignum, 16)
	return thelutil.Bytes2Hex(bignum_bytes)
}

// takes a string, converts to bytes, returns hex
func SHA3(tohash string) string {
	h := crypto.Sha3([]byte(tohash))
	return thelutil.Bytes2Hex(h)
}

// pack data into acceptable format for transaction
// TODO: make sure this is ok ...
// TODO: this is in two places, clean it up you putz
func PackTxDataArgs(args ...string) string {
	//fmt.Println("pack data:", args)
	ret := *new([]byte)
	for _, s := range args {
		if s[:2] == "0x" {
			t := s[2:]
			if len(t)%2 == 1 {
				t = "0" + t
			}
			x := thelutil.Hex2Bytes(t)
			//fmt.Println(x)
			l := len(x)
			ret = append(ret, thelutil.LeftPadBytes(x, 32*((l+31)/32))...)
		} else {
			x := []byte(s)
			l := len(x)
			// TODO: just changed from right to left. yabadabadoooooo take care!
			ret = append(ret, thelutil.LeftPadBytes(x, 32*((l+31)/32))...)
		}
	}
	return "0x" + thelutil.Bytes2Hex(ret)
	// return ret
}

// convert thelonious block to modules block
func convertBlock(block *types.Block) *epmtypes.Block {
	if block == nil {
		return nil
	}
	b := &epmtypes.Block{}
	b.Coinbase = hex.EncodeToString(block.Coinbase())
	b.Difficulty = block.Difficulty().String()
	b.GasLimit = block.GasLimit().String()
	b.GasUsed = block.GasUsed().String()
	b.Hash = hex.EncodeToString(block.Hash())
	//b.MinGasPrice = block.MinGasPrice.String()
	b.Nonce = hex.EncodeToString(block.Nonce())
	b.Number = block.Number().String()
	b.PrevHash = hex.EncodeToString(block.ParentHash())
	b.Time = int(block.Time())
	txs := make([]*epmtypes.Transaction, len(block.Transactions()))
	for idx, tx := range block.Transactions() {
		txs[idx] = convertTx(tx)
	}
	b.Transactions = txs
	b.TxRoot = hex.EncodeToString(block.TxHash())
	b.UncleRoot = hex.EncodeToString(block.UncleHash())
	b.Uncles = make([]string, len(block.Uncles()))
	for idx, u := range block.Uncles() {
		b.Uncles[idx] = hex.EncodeToString(u.Hash())
	}
	return b
}

// convert thelonious tx to modules tx
func convertTx(monkTx *types.Transaction) *epmtypes.Transaction {
	tx := &epmtypes.Transaction{}
	tx.ContractCreation = types.IsContractAddr(monkTx.To())
	tx.Gas = monkTx.Gas().String()
	tx.GasCost = monkTx.GasPrice().String()
	tx.Hash = hex.EncodeToString(monkTx.Hash())
	tx.Nonce = fmt.Sprintf("%d", monkTx.Nonce)
	tx.Recipient = hex.EncodeToString(monkTx.To())
	tx.Sender = hex.EncodeToString(monkTx.From())
	tx.Value = monkTx.Value().String()
	return tx
}

func PrettyPrintAccount(obj *monkstate.StateObject) {
	fmt.Println("Address", thelutil.Bytes2Hex(obj.Address())) //thelutil.Bytes2Hex([]byte(addr)))
	fmt.Println("\tNonce", obj.Nonce)
	fmt.Println("\tBalance", obj.Balance)
	if true { // only if contract, but how?!
		fmt.Println("\tInit", thelutil.Bytes2Hex(obj.InitCode))
		fmt.Println("\tCode", thelutil.Bytes2Hex(obj.Code))
		fmt.Println("\tStorage:")
		obj.EachStorage(func(key string, val *thelutil.Value) {
			val.Decode()
			fmt.Println("\t\t", thelutil.Bytes2Hex([]byte(key)), "\t:\t", thelutil.Bytes2Hex([]byte(val.Str())))
		})
	}
}

/*
// print all accounts and storage in a block
func PrettyPrintBlockAccounts(block *types.Block) {
	state := block.State()
	it := state.Trie.NewIterator()
	it.Each(func(key string, value *thelutil.Value) {
		addr := thelutil.Address([]byte(key))
		//        obj := monkstate.NewStateObjectFromBytes(addr, value.Bytes())
		obj := block.State().GetAccount(addr)
		PrettyPrintAccount(obj)
	})
}

// print all accounts and storage in the latest block
func PrettyPrintChainAccounts(mod *MonkModule) {
	curchain := mod.monk.thelonious.ChainManager()
	block := curchain.CurrentBlock()
	PrettyPrintBlockAccounts(block)
}*/
