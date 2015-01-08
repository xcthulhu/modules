esl.llll = {
	"name" : "LLLinkedList"

	//Constants
	"TailSlotOffset"  : "0"
	"HeadSlotOffset"  : "1"
	"LenSlotOffset"   : "2"

	"LLLLSlotSize" 	  : "3"

	"EntryMainOffset" : "0"
	"EntryPrevOffset" : "1"
	"EntryNextOffset" : "2"

	//Structure
	"TailSlot" : function(base){
		return Add(base, this.TailSlotOffset);
	},
	"HeadSlot" : function(base){
		return Add(base, this.HeadSlotOffset);
	},
	"LenSlot" : function(base){
		return Add(base, this.LenSlotOffset);
	},

	"MainSlot" : function(slot){
		return Add(slot, this.EntryMainOffset);
	},
	"PrevSlot" : function(slot){
		return Add(slot, this.EntryPrevOffset);
	},
	"NextSlot" : function(slot){
		return Add(slot, this.EntryNextOffset);
	},

	//Gets
	"Tail" : function(addr, base){
		return GetStorageAt(addr, this.TailSlot(base));
	},
	"Head" : function(addr, base){
		return GetStorageAt(addr, this.HeadSlot(base));
	},
	"Len"  : function(addr, base){
		return GetStorageAt(addr, this.LenSlot(base));
	}

	"Main" : function(addr, slot){
		return GetStorageAt(addr, this.MainSlot(slot));
	},
	"Prev" : function(addr, slot){
		return GetStorageAt(addr, this.PrevSlot(slot));
	},
	"Next" : function(addr, slot){
		return GetStorageAt(addr, this.NextSlot(slot));
	}

}