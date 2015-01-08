package ipfs

import(
	"github.com/eris-ltd/decerver-interfaces/core"
	"github.com/eris-ltd/decerver-interfaces/events"
	"github.com/eris-ltd/decerver-interfaces/modules"
	"github.com/eris-ltd/modules/ipfs/impl"
)

// implements decerver-interface module.
type(
	
	// This is the module.
	IpfsModule struct {
		ipfs   *impl.Ipfs
		ipfsApi *IpfsApi
	}

	// This is the api.
	IpfsApi struct {
		ipfs   *impl.Ipfs
	}
)

func NewIpfsModule() *IpfsModule {
	ipfs := &impl.Ipfs{}
	return &IpfsModule{ipfs, &IpfsApi{ipfs}}
}

func (mod *IpfsModule) Register(fileIO core.FileIO, rm core.RuntimeManager, eReg events.EventRegistry) error {
	rm.RegisterApiObject("ipfs", mod.ipfsApi)
	return nil
}

func (mod *IpfsModule) Init() error {
	return mod.ipfs.Init()
}

func (mod *IpfsModule) Start() error {
	mod.ipfs.Start()
	return nil
}

func (mod *IpfsModule) Shutdown() error {
	return mod.ipfs.Shutdown()
}

// TODO figure out when this would actually be used.
func (mod *IpfsModule) Restart() error {
	err := mod.Shutdown()
	if err != nil {
		return nil
	}
	return mod.Start()
}

func (mod *IpfsModule) SetProperty(name string, data interface{}) {
}

func (mod *IpfsModule) Property(name string) interface{} {
	return nil
}

func (mod *IpfsModule) ReadConfig(config_file string) {
}

func (mod *IpfsModule) WriteConfig(config_file string) {
}

func (mod *IpfsModule) Name() string {
	return "ipfs"
}

func (mod *IpfsModule) Subscribe(name string, event string, target string) chan events.Event {
	return nil
}

func (mod *IpfsModule) UnSubscribe(name string) {
	
}

func (api *IpfsApi) Get(cmd string, params ...string) modules.JsObject {
	return modules.JsReturnVal(api.ipfs.Get(cmd, params...))
}

func (api *IpfsApi) Push(cmd string, params ...string) modules.JsObject {
	return modules.JsReturnVal(api.ipfs.Push(cmd, params...))
}

func (api *IpfsApi) GetBlock(hash string) modules.JsObject {
	data, err := api.ipfs.GetBlock(hash)
	if err != nil {
		modules.JsReturnValErr(err)
	}
	return modules.JsReturnValNoErr(string(data))
}

func (api *IpfsApi) GetFile(hash string) modules.JsObject {
	data, err := api.ipfs.GetFile(hash)
	if err != nil {
		modules.JsReturnValErr(err)
	}
	return modules.JsReturnValNoErr(string(data))
}

// func (mod *IpfsModule) GetStream(hash string) (chan []byte, error) {
//	return modules.JsReturnVal(mod.ipfs.GetStream(hash))
// }

func (api *IpfsApi) GetTree(hash string, depth int) modules.JsObject {
	return modules.JsReturnVal(api.ipfs.GetTree(hash, depth))
}

func (api *IpfsApi) PushBlock(block []byte) modules.JsObject {
	return modules.JsReturnVal(api.ipfs.PushBlock(block))
}

func (api *IpfsApi) PushBlockString(block string) modules.JsObject {
	return modules.JsReturnVal(api.ipfs.PushBlockString(block))
}

func (api *IpfsApi) PushFile(fpath string) modules.JsObject {
	return modules.JsReturnVal(api.ipfs.PushFile(fpath))
}

func (api *IpfsApi) PushTree(fpath string, depth int) modules.JsObject {
	return modules.JsReturnVal(api.ipfs.PushTree(fpath, depth))
}

// IpfsModule should satisfy KeyManager
func (api *IpfsApi) ActiveAddress() modules.JsObject {
	return modules.JsReturnVal(api.ipfs.ActiveAddress(), nil)
}

func (api *IpfsModule) Addresses() modules.JsObject {
	count := api.ipfs.AddressCount()
	addresses := make(modules.JsObject)
	array := make([]string, count)

	for i := 0; i < count; i++ {
		addr, _ := api.ipfs.Address(i)
		array[i] = addr
	}
	addresses["Addresses"] = array
	return modules.JsReturnVal(addresses, nil)
}

func (api *IpfsApi) Address(n int) modules.JsObject {
	return modules.JsReturnVal(api.ipfs.Address(n))
}

func (api *IpfsApi) SetAddress(addr string) modules.JsObject {
	err := api.ipfs.SetAddress(addr)
	if err != nil {
		return modules.JsReturnValErr(err)
	} else {
		// No error means success.
		return modules.JsReturnValNoErr(nil)
	}
}

func (api *IpfsApi) SetAddressN(n int) modules.JsObject {
	return modules.JsReturnVal(nil, api.ipfs.SetAddressN(n))
}

func (api *IpfsApi) NewAddress(set bool) modules.JsObject {
	return modules.JsReturnVal(api.ipfs.NewAddress(set))
}

func (api *IpfsApi) AddressCount() modules.JsObject {
	return modules.JsReturnValNoErr(api.ipfs.AddressCount())
}
