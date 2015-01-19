package monk

import (
	"fmt"
	"github.com/eris-ltd/new-thelonious/thelutil"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"testing"
	"time"
)

/*
   TestSimpleStorage
   TestMsgStorage
   TestTx
   TestManyTx
*/

// called by `go test` functions
func tester(name string, testing func(mod *MonkModule), end int) {
	mod := NewMonk(nil)
	mod.Config.Mining = false
	mod.Config.LogLevel = 3
	mod.Config.DbMem = true
	//g := mod.LoadGenesis(mod.Config.GenesisConfig)
	//g.Difficulty = 10 // so we always mine quickly
	//mod.SetGenesis(g)

	testing(mod)

	if end > 0 {
		time.Sleep(time.Second * time.Duration(end))
	}
	//PrettyPrintChainAccounts(mod)
	mod.Shutdown()
	time.Sleep(time.Second * 3)
}

// compare expected and recovered vals
func check_recovered(expected, recovered string) bool {
	if thelutil.Coerce2Hex(recovered) == thelutil.Coerce2Hex(expected) {
		fmt.Println("Test passed")
		return true
	} else {
		fmt.Println("Test failed. Expected", expected, "Recovered", recovered)
		return false
	}
}

// contract that stores a single value during init
func TestSimpleStorage(t *testing.T) {
	tester("simple storage", func(mod *MonkModule) {
		mod.Init()
		// set up test parameters and code
		key := "0x5"
		value := "0x400"
		code := fmt.Sprintf(`
            {
                ;; store a value
                [[%s]]%s
            }
        `, key, value)
		fmt.Println("Code:\n", code)
		// write code to file and deploy
		c := "tests/simple-storage.lll"
		p := path.Join(mod.monk.config.ContractPath, c)
		err := ioutil.WriteFile(p, []byte(code), 0644)
		if err != nil {
			fmt.Println("write file failed", err)
			os.Exit(0)
		}
		contract_addr, err := mod.Script(p, "lll")
		if err != nil {
			t.Fatal(err)
		}
		mod.Start()
		mod.Commit()

		recovered := "0x" + mod.StorageAt(contract_addr, key)
		result := check_recovered(value, recovered)
		if !result {
			t.Error("got:", recovered, "expected:", value)
		}
	}, 0)
}

// test a simple key-value store contract
func TestMsgStorage(t *testing.T) {
	tester("msg storage", func(mod *MonkModule) {
		mod.Init()
		contract_addr, err := mod.Script(path.Join(mod.Config.ContractPath, "tests/keyval.lll"), "lll")
		if err != nil {
			t.Fatal(err)
		}
		mod.Start()
		mod.Commit()

		key := "0x21"
		value := "0x400"
		time.Sleep(time.Nanosecond) // needed or else subscribe channels block and are skipped ... TODO: why?!
		//fmt.Println("contract account:", mod.Account(contract_addr))
		//fmt.Println("my account:", mod.Account(mod.ActiveAddress()))

		mod.Msg(contract_addr, []string{key, value})

		mod.Commit()

		start := time.Now()
		recovered := "0x" + mod.StorageAt(contract_addr, key)
		dif := time.Since(start)
		fmt.Println("get storage took", dif)
		result := check_recovered(value, recovered)
		if !result {
			t.Error("got:", recovered, "expected:", value)
		}

	}, 0)
}

// test simple tx
func TestTx(t *testing.T) {
	tester("basic tx", func(mod *MonkModule) {
		mod.Init()
		addr := "b9398794cafb108622b07d9a01ecbed3857592d5"
		addr_bytes := thelutil.Hex2Bytes(addr)
		amount := "567890"
		old_balance := mod.monk.pipe.Balance(addr_bytes)
		//mod.SetCursor(0)
		start := time.Now()
		mod.Tx(addr, amount)
		dif := time.Since(start)
		fmt.Println("sending one tx took", dif)
		mod.Start()
		mod.Commit()

		new_balance := mod.monk.pipe.Balance(addr_bytes)
		old := old_balance.BigInt()
		am := thelutil.Big(amount)
		n := new(big.Int)
		n.Add(old, am)
		newb := thelutil.BigD(new_balance.Bytes())
		//t.success = check_recovered(n.String(), newb.String())
		result := check_recovered(n.String(), newb.String())
		if !result {
			t.Error("got:", newb.String(), "expected:", n.String())
		}
	}, 0)
}

// test tx with gas etc.
func TestTransaction(t *testing.T) {
	tester("basic tx", func(mod *MonkModule) {
		mod.Init()
		addr := "b9398794cafb108622b07d9a01ecbed3857592d5"
		amount := "567890"
		mod.Transact(addr, amount, "1000000", "100000", "")
		mod.Start()
		mod.Commit()
	}, 0)
}

func TestManyTx(t *testing.T) {
	tester("many tx", func(mod *MonkModule) {
		mod.Init()
		addr := "b9398794cafb108622b07d9a01ecbed3857592d5"
		addr_bytes := thelutil.Hex2Bytes(addr)
		amount := "567890"
		old_balance := mod.monk.pipe.Balance(addr_bytes)
		N := 1000
		//mod.SetCursor(0)
		start := time.Now()
		for i := 0; i < N; i++ {
			mod.Tx(addr, amount)
		}
		end := time.Since(start)
		fmt.Printf("sending %d txs took %s\n", N, end)
		mod.Start()
		mod.Commit()

		new_balance := mod.monk.pipe.Balance(addr_bytes)
		old := old_balance.BigInt()
		am := thelutil.Big(amount)
		mult := big.NewInt(int64(N))
		n := new(big.Int)
		n.Add(old, n.Mul(mult, am))
		newb := thelutil.BigD(new_balance.Bytes())
		results := check_recovered(n.String(), newb.String())
		if !results {
			t.Error("got:", newb.String(), "expected:", n.String())
		}

	}, 0)
}

func TestSetProperty(t *testing.T) {
	m := NewMonk(nil)
	value := "somechainid"
	m.SetProperty("ChainId", value)
	m.Init()
	if m.Config.ChainId != value {
		t.Error("got:", m.Config.ChainId, "expected:", value)
	}

}

/*
func receiveModule(m modules.Module) {
}

func receiveBlockchain(m modules.Blockchain) {
}

// Static type checking to ensure the module and blockchain interfaces are satisfied
func TestModule(t *testing.T) {
	tester("module satisfaction", func(mod *MonkModule) {
		receiveModule(mod)
		receiveBlockchain(mod)
		receiveBlockchain(mod.monk)
	}, 0)
}

func TestSubscribe(t *testing.T) {
	tester("subscribe/unsuscribe", func(mod *MonkModule) {
		mod.Init()
		name := "testNewBlock"
		ch := mod.Subscribe(name, "newBlock", "")
		go func() {
			for {
				a, more := <-ch
				if !more {
					break
				}
				if _, ok := a.Resource.(*modules.Block); !ok {
					t.Error("Event resource not a block!")
				}
			}
		}()
		mod.Start()
		time.Sleep(4 * time.Second)
		mod.UnSubscribe("testNewBlock")
	}, 0)
}*/
