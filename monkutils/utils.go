package monkutils

import (
	"fmt"
	"os"

	"github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/thelonious/monkdb"
	"github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/thelonious/monkwire"
)

/*
   ********** WARNING ************
   THESE FUNCTIONS WILL FAIL ON ERR
   ********************************
*/

func NewDatabase(dbName string, mem bool) monkutil.Database {
	if mem {
		db, err := monkdb.NewMemDatabase()
		if err != nil {
			exit(err)
		}
		return db
	}
	db, err := monkdb.NewLDBDatabase(dbName)
	if err != nil {
		exit(err)
	}
	return db
}

func NewClientIdentity(clientIdentifier, version, customIdentifier string) *monkwire.SimpleClientIdentity {
	return monkwire.NewSimpleClientIdentity(clientIdentifier, version, customIdentifier)
}

func NewKeyManager(KeyStore string, Datadir string, db monkutil.Database) *monkcrypto.KeyManager {
	var keyManager *monkcrypto.KeyManager
	switch {
	case KeyStore == "db":
		keyManager = monkcrypto.NewDBKeyManager(db)
	case KeyStore == "file":
		keyManager = monkcrypto.NewFileKeyManager(Datadir)
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
func CompileLLL(filename string, literal bool) string {
	code, err := monkutil.CompileLLL(filename, literal)
	if err != nil {
		fmt.Println("error compiling lll!", err)
		return ""
	}
	return monkutil.Bytes2Hex(code)
}
