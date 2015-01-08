package legalmarkdown

import (
	"fmt"
	"github.com/eris-ltd/decerver-interfaces/core"
	"github.com/eris-ltd/decerver-interfaces/events"
	"github.com/eris-ltd/decerver-interfaces/modules"
	"github.com/eris-ltd/legalmarkdown/lmd"
)

type LmdApi struct {
	name string
}

// pass two strings after reading a template file and an optional
// params file. the params file can be an empty string if the
// parameters are contained within the lmd template file that
// is read into the contents string.
//
// the returned string will be a PDF which can be written or
// displayed by an PDF reader.
func (lmda *LmdApi) Compile(contents, params string) modules.JsObject {
	fmt.Println("Contents: " + contents)
	fmt.Println("Params: " + params)
	res := lmd.RawMarkdownToPDF(contents, params)
	fmt.Println("Nice... " + res)

	//return modules.JsReturnValNoErr(res);
	return nil
}

// implements decerver-interface module
type LmdModule struct {
	api *LmdApi
}

func NewLmdModule() *LmdModule {
	lmdApi := &LmdApi{}
	return &LmdModule{lmdApi}
}

func (mod *LmdModule) Register(fileIO core.FileIO, rm core.RuntimeManager, eReg events.EventRegistry) error {
	rm.RegisterApiObject("lmd", mod.api)
	return nil
}

func (mod *LmdModule) Init() error {
	return nil
}

func (mod *LmdModule) Start() error {
	return nil
}

func (mod *LmdModule) Shutdown() error {
	return nil
}

func (mod *LmdModule) Restart() error {
	return nil
}

func (mod *LmdModule) SetProperty(name string, data interface{}) {
}

func (mod *LmdModule) Property(name string) interface{} {
	return nil
}

func (mod *LmdModule) ReadConfig(config_file string) {
}

func (mod *LmdModule) WriteConfig(config_file string) {
}

func (mod *LmdModule) Name() string {
	return "lmd"
}

func (mod *LmdModule) Subscribe(name string, event string, target string) chan events.Event {
	return nil
}

func (mod *LmdModule) UnSubscribe(name string) {
}
