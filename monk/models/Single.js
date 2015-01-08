esl.single = {
	"name" : "Single",

	//Structure
	"ValueSlot" : function(varname){
		return esl.stdvar.VarBase(esl.stdvar.Vari(varname));
	},

	//Gets
	"Value" : function(addr, varname){
		return GetStorageAt(addr, this.ValueSlot(varname));
	},
}