package thelutils

import (
	"fmt"
	"os"

	"github.com/eris-ltd/lllc-server"
	"github.com/eris-ltd/new-thelonious/crypto"
	"github.com/eris-ltd/new-thelonious/theldb"
	"github.com/eris-ltd/new-thelonious/thelutil"
	//"github.com/eris-ltd/new-thelonious/monkwire"
)

/*
   ********** WARNING ************
   THESE FUNCTIONS WILL FAIL ON ERR
   ********************************
*/

func NewDatabase(dbName string, mem bool) thelutil.Database {
	if mem {
		db, err := theldb.NewMemDatabase()
		if err != nil {
			exit(err)
		}
		return db
	}
	db, err := theldb.NewLDBDatabase(dbName)
	if err != nil {
		exit(err)
	}
	return db
}

/*func NewClientIdentity(clientIdentifier, version, customIdentifier string) *monkwire.SimpleClientIdentity {
	return monkwire.NewSimpleClientIdentity(clientIdentifier, version, customIdentifier)
}*/

func NewKeyManager(KeyStore string, Datadir string, db thelutil.Database) *crypto.KeyManager {
	var keyManager *crypto.KeyManager
	switch {
	case KeyStore == "db":
		keyManager = crypto.NewDBKeyManager(db)
	case KeyStore == "file":
		keyManager = crypto.NewFileKeyManager(Datadir)
	default:
		exit(fmt.Errorf("unknown keystore type: %s", KeyStore))
	}
	return keyManager
}

func exit(err error) {
	status := 0
	if err != nil {
		fmt.Println(err)
		status = 1
	}
	os.Exit(status)
}

// compile LLL file into evm bytecode
// returns hex
func Compile(filename string) string {
	code, err := lllcserver.Compile(filename)
	if err != nil {
		fmt.Println("error compiling lll!", err)
		return ""
	}
	return thelutil.Bytes2Hex(code)
}
