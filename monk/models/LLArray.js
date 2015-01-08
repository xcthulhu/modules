esl.llarray = {
	"name" : "LLArray",

	//Constants
	"ESizeOffset" : "0",

	"MaxEOffset" : "0",
	"StartOffset" : "1",

	//Structure
	"ESizeslot" : function(base){
		return Add(base, this.ESizeOffset);
	},
	"Maxeslot" : function(slot){
		return Add(slot, this.MaxEOffset);
	},
	"StartSlot" : function(slot){
		return Add(slot, this.StartOffset);
	},

	//Gets
	"ESize" : function(addr, base){
		return GetStorageAt(addr, this.ESizeslot(base));
	},
	
	"MaxE" : function(addr, slot){
		return GetStorageAt(addr, this.Maxeslot(slot));
	},

	"Element" : function(addr, base, slot, index){
		var Esize = this.GetESize(addr, base);
		if(this.GetMaxE(addr, slot) < index){
			return "0";
		}

		if(Esize == "0x100"){
			return GetStorageAt(addr, Add(index, this.StartOffset));
		}else{
			var eps = Div("0x100",Esize);
			var pos = Mod(index, eps);
			var row = Add(Mod(Div(index, eps),"0xFFFF"), this.StartOffset);

			var sval = GetStorageAt(addr, row);
			return Mod(Div(sval, Exp(Esize, pos)), Exp("2", Esize)); 
		}
	},
}