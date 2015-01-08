esl.array = {
	"name" : "Array"
 
	//Structure
	"CTS" : function(name, key){
		return Add(esl.stdvar.Vari(name), Add(Mul(Mod(key, Exp("0x100", "20")), Exp("0x100", "3")), Exp("0x100","2"));
	},
	"CTK" : function(slot){
		return Mod(Div(slot, Exp("0x100","3")), Exp("0x100","20"));
	},

	"ESizeslot" : function(name){
		return esl.llarray.ESizeslot(esl.stdvar.VariBase(name));
	},
	"Maxeslot" : function(key){
		return esl.llarray.Maxeslot(this.CTS(name, key));
	},
	"StartSlot" : function(key){
		return esl.llarray.StartSlot(this.CTS(name, key));
	},

	//Gets
	"ESize" : function(addr, name){
		return esl.llarray.ESize(addr, esl.stdvar.VariBase(name));
	},
	
	"MaxE" : function(addr, name, key){
		return esl.llarray.MaxE(addr, this.CTS(name, key));
	},

	"Element" : function(addr, name, key, index){
		return esl.llarray.Element(addr, esl.stdvar.Vari(name), this.CTS(name, key), index)
	},
}