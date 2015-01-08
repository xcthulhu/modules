package blockchaininfo

import (
	"fmt"
	"github.com/eris-ltd/decerver-interfaces/events"
	"github.com/eris-ltd/decerver-interfaces/modules"
	"testing"
	"time"
)

var (
	BlockChainInfo = start()
	blockHash      = "000000000000000016d65758ed8df787c3d490c569578d38d6db2ed4b56817f0"
	acct1          = "15v4EdEsnt367mgUdqSvbS7xExXTwKWoTo"
	acct2          = ""
	acct3          = ""
	guid           = ""
	passwd1        = ""
	passwd2        = ""
	apicode        = ""
)

func start() *BlkChainInfo {
	b := NewBlkChainInfo()
	_ = b.Init()

	b.BciApi.GUID = guid
	b.BciApi.Password = passwd1
	b.BciApi.SecondPassword = passwd2
	b.BciApi.APICode = apicode

	return b
}

func testBlockEquality(block *modules.Block) error {
	if block.Number != "329896" {
		return fmt.Errorf("Block number is not right. Expected: 329896, Got: %s", block.Number)
	}

	if block.Time != 1415922366 {
		return fmt.Errorf("Block time is not right. Expected: %v, Got: %v", 1322131230, block.Time)
	}

	if block.Nonce != "2245627664" {
		return fmt.Errorf("Block nonce is not right. Expected: %s, Got: %s", "2964215930", block.Nonce)
	}

	if block.Hash != "000000000000000016d65758ed8df787c3d490c569578d38d6db2ed4b56817f0" {
		return fmt.Errorf("Go Kill Yourself. The blockhash searched on does not equal the blockhash returned.")
	}

	if block.PrevHash != "0000000000000000168017e70167b30132ee606e99fbbfc6bf7d0dcb0388286c" {
		return fmt.Errorf("Block previous hash is not right. Expected: %s, Got: %s", "0000000000000000168017e70167b30132ee606e99fbbfc6bf7d0dcb0388286c", block.PrevHash)
	}

	if block.TxRoot != "d3cee9d795cdee08ea36aeee2c2a481b2beb092d12d345c01134ab21d48d910f" {
		return fmt.Errorf("Block previous hash is not right. Expected: %s, Got: %s", "d3cee9d795cdee08ea36aeee2c2a481b2beb092d12d345c01134ab21d48d910f", block.TxRoot)
	}
	return nil
}

func testTxEquality(tx *modules.Transaction) error {
	return nil
}

func TestTx(t *testing.T) {
	hash, err := BlockChainInfo.Tx(acct3, "500")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Successful test transfer: ", hash)
}

func TestBlock(t *testing.T) {
	block := BlockChainInfo.Block(blockHash)
	err := testBlockEquality(block)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLatestBlock(t *testing.T) {
	latestBlock := BlockChainInfo.LatestBlock()
	if len(latestBlock) != 64 {
		t.Fatal("Latest block hash is incorrect.")
	}
}

func TestBlockHeight(t *testing.T) {
	blockHeight := BlockChainInfo.BlockCount()
	if blockHeight <= 329917 {
		t.Fatal("Block height is incorrect.")
	}
}

func TestAccount(t *testing.T) {
	acct1Res := BlockChainInfo.Account(acct1)
	if acct1Res.Balance != "0" {
		t.Fatalf("Incorrect balance. Expected: %s, Got: %s. Check https://blockchain.info/address/15v4EdEsnt367mgUdqSvbS7xExXTwKWoTo first.", 0, acct1Res.Balance)
	}
	if acct1Res.Nonce != "2" {
		t.Fatalf("Incorrect nonce. Expected: %s, Got: %s. Check https://blockchain.info/address/15v4EdEsnt367mgUdqSvbS7xExXTwKWoTo first.", 2, acct1Res.Nonce)
	}
}

// Note these will take time so not ideal to run them all the time.
func TestBlockPolling(t *testing.T) {
	fmt.Println("*** Hold Fast. Long Tests coming. This test should be run with: go test -timeout=13m")
	ch := BlockChainInfo.Subscribe("newBlock", "", "")
	go dropBox(ch)
	interval, _ := time.ParseDuration("12m")
	time.Sleep(interval)
	BlockChainInfo.UnSubscribe("newBlock")
}

// You'll have to manually send or receive from the acct2 address
func TestSingleAddressPolling(t *testing.T) {
	ch := BlockChainInfo.Subscribe("addr", "tx", acct2)
	go dropBox(ch)
	interval, _ := time.ParseDuration("5m")
	time.Sleep(interval)
	BlockChainInfo.UnSubscribe(acct2)
}

func TestMultAddressPolling(t *testing.T) {
	ch1 := BlockChainInfo.Subscribe("addr", "tx", acct2)
	go dropBox(ch1)
	ch2 := BlockChainInfo.Subscribe("addr", "tx", acct3)
	go dropBox(ch2)
	interval, _ := time.ParseDuration("5m")
	time.Sleep(interval)
	BlockChainInfo.UnSubscribe(acct3)
	BlockChainInfo.UnSubscribe(acct2)
}

func dropBox(ch chan events.Event) {
	fmt.Println(<-ch)
}
