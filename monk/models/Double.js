esl.double = {
	"name" : "Double",

	//Structure
	"ValueSlot" : function(varname){
		return Add(stdvar.Vari(varname),stdvar.VarSlotSize);
	},
	"ValueSlot2" : function(varname){
		return Add(this.ValueSlot(varname),1);
	},

	//Gets
	"Value" : function(addr, varname){
		var values = [];
		values.push(GetStorageAt(addr, this.ValueSlot(varname)));
		values.push(GetStorageAt(addr, this.ValueSlot2(varname)));
		return values
	},

}