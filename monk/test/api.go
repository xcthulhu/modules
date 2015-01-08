package test

import (
	"github.com/eris_ltd/decerver-interfaces/core"
	"github.com/eris_ltd/modules/monk"
)

func TestApi(rt *core.Runtime){
	mjs := monkjs.NewMonkJs()
	mjs.restart()
	rt.BindScriptObject("monk",mjs)
	
}

var js string = `

	function runtests(){
		testStorageAt();
	};
	
	function testStorageAt(){
		var sa = monk.StorageAt("0x0","0x0");
		Println("sa");
	};

`;