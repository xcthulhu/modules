package monkpipe

import "github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"

var cnfCtr = monkutil.Hex2Bytes("661005d2720d855f1d9976f88bb10c1a3398c77f")

type Config struct {
	pipe *Pipe
}

func (self *Config) Get(name string) *Object {
	configCtrl := self.pipe.World().safeGet(cnfCtr)
	var addr []byte

	switch name {
	case "NameReg":
		addr = []byte{0}
	case "DnsReg":
		objectAddr := configCtrl.GetStorage(monkutil.BigD([]byte{0}))
		domainAddr := (&Object{self.pipe.World().safeGet(objectAddr.Bytes())}).StorageString("DnsReg").Bytes()
		return &Object{self.pipe.World().safeGet(domainAddr)}
	default:
		addr = monkutil.RightPadBytes([]byte(name), 32)
	}

	objectAddr := configCtrl.GetStorage(monkutil.BigD(addr))

	return &Object{self.pipe.World().safeGet(objectAddr.Bytes())}
}

func (self *Config) Exist() bool {
	return self.pipe.World().Get(cnfCtr) != nil
}
