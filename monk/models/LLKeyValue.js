esl.llkv = {
	"name" 	: "LLKeyValue",

	//Functions
	"Value" : function(addr, slot, offset){
		return GetStorageAt(addr, Add(slot, offset));
	},
}