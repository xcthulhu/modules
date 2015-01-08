esl.stdvar = {
	"name" : "StdVarSpace",

	//Constants
	"VarSlotSize" 	: "0x5"
	"StdVarOffset" 	: "0x1"

	//Functions?
	"Vari" 	: function(varname){
		return Add(Mul(NSBase, this.StdVarOffset)),Mul(Div(SHA3(varname),Exp("0x100", "24")),Exp("0x100", "23"));
	},
	"VarBase" 	: function(varname){
		return Add(varname, this.VarSlotSize)
	},
	"VariBase" : function(varname){
		return this.VarBase(this.Vari(varname))
	},

	//Data Slots
	"VarTypeSlot"	: function(varname){
		return this.Vari(varname);
	},
	"VarNameSlot"	: function(varname){
		return Add(this.Vari(varname), 1);
	},
	"VarAddpermSlot"	: function(varname){
		return Add(this.Vari(varname), 2);
	},
	"VarRmpermSlot" 	: function(varname){
		return Add(this.Vari(varname), 3);
	},
	"VarModpermSlot"	: function(varname){
		return Add(this.Vari(varname), 4);
	},

	//Getting Variable stuff
	"Type" 	: function(addr, varname){
		return GetStorageAt(addr,this.VarTypeSlot);
	},
	"Name" 	: function(addr, varname){
		return GetStorageAt(addr,this.VarNameSlot);
	},
	"Addperm" 	: function(addr, varname){
		return GetStorageAt(addr,this.VarAddPermSlot);
	},
	"Rmperm" 	: function(addr, varname){
		return GetStorageAt(addr,this.VarRmPermSlot);
	},
	"Modperm" 	: function(addr, varname){
		return GetStorageAt(addr,this.VarModPermSlot);
	},
}