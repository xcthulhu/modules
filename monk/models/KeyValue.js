esl.keyvalue = {
	"name" 	: "KeyValue",

	//Constants
	//None

	"CTS" : function(name, key){
		return Add(esl.stdvar.Vari(name), Add(Mul(Mod(key, Exp("0x100", "20")), Exp("0x100", "3")), Exp("0x100","2"));
	},
	"CTK" : function(slot){
		return Mod(Div(slot, Exp("0x100","3")), Exp("0x100","20"));
	},
	
	"value" : function(addr, varname, key){
		return esl.llkv.Value(addr, this.CTS(varname, key), "0")
	},

}