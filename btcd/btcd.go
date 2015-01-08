package btcdglue

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/eris-ltd/decerver-interfaces/core"
	"github.com/eris-ltd/decerver-interfaces/events"
	"github.com/eris-ltd/decerver-interfaces/modules"

	rpc "github.com/conformal/btcrpcclient"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
)

type BTC struct {
	btcdConfig   *rpc.ConnConfig
	walletConfig *rpc.ConnConfig
	// btcrpcclient does not allow for new subscriptions
	//   once the client is already started
	// so we make a new client (websocket connection) for each subscription
	// and a corresponding channel to push things back up to the decerver
	// we also use a single client for get and push calls
	client   *rpc.Client
	notifies map[string]*rpc.Client
	chans    map[string]chan events.Event

	btcproc    *os.Process
	walletproc *os.Process
}

func (b *BTC) Register(fileIO core.FileIO, runtime core.Runtime, eReg events.EventRegistry) error {
	return nil
}

func NewBtcd() *BTC {
	return &BTC{}
}

func (b *BTC) Init() error {
	// hack to get rpc.cert
	cmd := exec.Command("btcd", "--nodnsseed", "-u", "rpcuser", "-P", "rpcpass")
	go cmd.Run()
	time.Sleep(2 * time.Second)
	cmd.Process.Kill()
	cmd = exec.Command("btcwallet", "-u", "rpcuser", "-P", "rpcpass")
	go cmd.Run()
	time.Sleep(2 * time.Second)
	cmd.Process.Kill()

	// setup config
	btcdHomeDir := btcutil.AppDataDir("btcd", false)
	certs, err := ioutil.ReadFile(filepath.Join(btcdHomeDir, "rpc.cert"))
	if err != nil {
		log.Fatal(err)
	}
	connCfg := &rpc.ConnConfig{
		Host:         "localhost:18556",
		Endpoint:     "ws",
		User:         "rpcuser", // load these from file ya
		Pass:         "rpcpass",
		Certificates: certs,
	}
	b.btcdConfig = connCfg

	btcdHomeDir = btcutil.AppDataDir("btcwallet", false)
	certs, err = ioutil.ReadFile(filepath.Join(btcdHomeDir, "rpc.cert"))
	if err != nil {
		log.Fatal(err)
	}
	connCfg = &rpc.ConnConfig{
		Host:         "localhost:18554",
		Endpoint:     "ws",
		User:         "rpcuser", // load these from file ya
		Pass:         "rpcpass",
		Certificates: certs,
	}
	b.walletConfig = connCfg

	b.chans = make(map[string]chan events.Event)
	b.notifies = make(map[string]*rpc.Client)

	return nil
}

func startProc(cmd *exec.Cmd, config *rpc.ConnConfig) error {
	cmd.Stdout = os.Stdout
	go cmd.Run()

	tries := 5
	timeout := time.Second
	var client *rpc.Client
	var err error = fmt.Errorf("notnill")
	i := 0
	for ; i < tries && err != nil; i++ {
		time.Sleep(timeout)
		client, err = rpc.New(config, nil)
	}
	if i == tries {
		log.Fatal(err)
	}
	client.Shutdown()
	return nil
}

func (b *BTC) Start() error {
	// start up btcd
	cmd := exec.Command("btcd", "--simnet", "--nodnsseed", "-u", "rpcuser", "-P", "rpcpass")
	startProc(cmd, b.btcdConfig)
	b.btcproc = cmd.Process
	// start the wallet server
	cmd = exec.Command("btcwallet", "--simnet", "-u", "rpcuser", "-P", "rpcpass")
	startProc(cmd, b.walletConfig)
	b.walletproc = cmd.Process

	// need to do some key stuff

	// start the websocket client for general gets
	// try X times to give it a chance to boot up, otherwise fail
	client, _ := rpc.New(b.walletConfig, nil)
	b.client = client

	// subscribe to new blocks
	//	b.Subscribe("newBlock", "newBlock", "")

	return nil
}

func (b *BTC) Shutdown() error {
	b.client.Shutdown()
	for _, c := range b.notifies {
		c.Shutdown()
	}
	//TODO: close channels

	// shutdown wallet and btcd
	b.walletproc.Signal(os.Interrupt)
	b.walletproc.Wait()
	b.btcproc.Signal(os.Interrupt)
	b.btcproc.Wait()
	return nil
}

func (b *BTC) ReadConfig(config_file string) {

}

func (b *BTC) WriteConfig(config_file string) {

}

func (b *BTC) Name() string {
	return "btcd"
}

func (b *BTC) Subscribe(name string, event string, target string) chan events.Event {
	// for each subscription we create a new websocket connection client
	// with a set of handlers (the callbacks)
	// and a corresponding channel to push the event up
	handlers := rpc.NotificationHandlers{}
	ch := make(chan events.Event)
	eve := events.Event{
		Event:     event,
		Target:    target,
		Source:    b.Name(),
		TimeStamp: time.Now(),
	}
	switch name {
	case "newBlock":
		handlers.OnBlockConnected = func(hash *btcwire.ShaHash, height int32) {
			eve.Resource = hash
			ch <- eve
		}
	case "tx":
		handlers.OnBlockConnected = func(hash *btcwire.ShaHash, height int32) {
			eve.Resource = hash
			ch <- eve
		}
	}
	client, err := rpc.New(b.walletConfig, &handlers)
	if err != nil {
		log.Println("cmah!!!", err)
		return nil
	}
	switch name {
	case "newBlock":
		client.NotifyBlocks()
	case "tx":

	}
	b.chans[name] = ch
	b.notifies[name] = client
	return ch
}

/*
   -------------
   "block" : hexHash
   "tx"    : hexHash
*/
func (b *BTC) Get(cmd string, params ...string) (ret interface{}, err error) {
	switch cmd {
	case "block":
		hash, _ := hex.DecodeString(params[0])
		shaHash, err := btcwire.NewShaHash(hash)
		if err != nil {
			return nil, err
		}
		//return (*btcutil.Block, error)
		ret, err = b.client.GetBlock(shaHash)
	case "block-count":
		//return (int64, error)
		ret, err = b.client.GetBlockCount()
	case "tx":
		hash, _ := hex.DecodeString(params[0])
		txHash, err := btcwire.NewShaHash(hash)
		if err != nil {
			return nil, err
		}
		// return (btcutil.Tx)
		ret, err = b.client.GetRawTransaction(txHash)
	case "npeers":
		//return (int64, error)
		ret, err = b.client.GetConnectionCount()
	case "accounts":
		ret, err = b.client.ListAccounts()
	case "newwallet":
		err = b.client.CreateEncryptedWallet(params[0])
	case "address":
		ret, err = b.client.GetAccountAddress("")
	}
	return
}

func (b *BTC) Push(cmd string, params ...string) (ret string, err error) {
	/*
	   switch(cmd){
	       case "tx":
	           //  func (c *Client) SendFrom(fromAccount string, toAddress btcutil.Address, amount btcutil.Amount) (*btcwire.ShaHash, error)
	           from := params[0]
	           to := params[1]
	           amount := params[2]

	          // func DecodeAddress(addr string, defaultNet *btcnet.Params) (Address, error)
	           addr, err := btcutil.Decodeaddress(to, 0)
	           if err != nil{
	               return "", err
	           }
	           ret, err = b.client.SendFrom(from,

	   }*/
	return
}

func (b *BTC) Commit() {
	b.client.SetGenerate(true, 1) // num cpus
	_ = <-b.chans["newBlock"]
	b.client.SetGenerate(false, 1)
}

func (b *BTC) AutoCommit(toggle bool) error {
	return b.client.SetGenerate(toggle, 1) // num cpus
}

func (b *BTC) IsAutocommit() bool {
	ret, _ := b.client.GetGenerate()
	return ret
}

/*
   BTCD does not yet have support for mining and account balances
*/

func (b *BTC) State() modules.State {
	// not currently supported for btcd
	return modules.State{}
}

func (b *BTC) Storage(target string) modules.Storage {
	// not currently supported for btcd
	return modules.Storage{}
}
