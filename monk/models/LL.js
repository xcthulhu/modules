esl.ll = {
	"name" : "LinkedList"

	//Structure
	"CTS" : function(name, key){
		return Add(esl.stdvar.Vari(name), Add(Mul(Mod(key, Exp("0x100", "20")), Exp("0x100", "3")), Exp("0x100","2"));
	},
	"CTK" : function(slot){
		return Mod(Div(slot, Exp("0x100","3")), Exp("0x100","20"));
	},

	"TailSlot" : function(name){
		return esl.llll.TailSlot(esl.stdvar.VariBase(name));
	},
	"HeadSlot" : function(name){
		return esl.llll.HeadSlot(esl.stdvar.VariBase(name));
	},
	"LenSlot" : function(name){
		return esl.llll.LenSlot(esl.stdvar.VariBase(name));
	},

	"MainSlot" : function(name, key){
		return esl.llll.MainSlot(this.CTS(name, key));
	},
	"PrevSlot" : function(name, key){
		return esl.llll.Prevlot(this.CTS(name, key));
	},
	"NextSlot" : function(name, key){
		return esl.llll.NextSlot(this.CTS(name, key));
	},

	//Gets
	"TailAddr" : function(addr, name){
		tail=GetStorageAt(addr, this.TailSlot(name));
		if(tail=="0"){
			return null;
		}
		else{
			return tail;
		}
	},
	"HeadAddr" : function(addr, name){
		head=GetStorageAt(addr, this.HeadSlot(name));
		if(head=="0"){
			return null;
		}
		else{
			return head;
		}
	},
	"Tail" : function(addr, name){
		head=GetStorageAt(addr, this.HeadSlot(name));
		if(head=="0"){
			return null;
		}
		else{
			return this.CTK(tail);
		}
	},
	"Head" : function(addr, name){
		head=GetStorageAt(addr, this.HeadSlot(name));
		if(head=="0"){
			return null;
		}
		else{
			return this.CTK(head);
		}
	},
	"Len"  : function(addr, name){
		return GetStorageAt(addr, this.LenSlot(name));
	},

	"Main" : function(addr, name, key){
		return GetStorageAt(addr, this.MainSlot(name, key));
	},
	"PrevAddr" : function(addr, name, key){
		prev=GetStorageAt(addr, this.PrevSlot(name, key));
		if(prev==="0"){
			return null;
		}
		else{
			return prev;
		}
	},
	"NextAddr" : function(addr, name, key){
		next=GetStorageAt(addr, this.NextSlot(name, key));
		if(next==="0"){
			return null;
		}
		else{
			return next;
		}
	},
	"Prev" : function(addr, name, key){
		prev=GetStorageAt(addr, this.PrevSlot(name, key));
		if(prev==="0"){
			return null;
		}
		else{
			return this.CTK(prev);
		}	
	},
	"Next" : function(addr, name, key){
		next=GetStorageAt(addr, this.NextSlot(name, key));
		if(next==="0"){
			return null;
		}
		else{
			return this.CTK(next);
		}
	},

	//Gets the whole list. Note the separate function which gets the keys
	"GetList" : function(addr, name){
		var list = [];
		var current = this.Tail(addr, name);
		while(current!=null){
			list.push(this.Main(addr, current));
			current = this.Next(addr, current);
		}

		return list;
	},

	"GetKeys" : function(addr, name){
		var keys = [];
		var current = this.Tail(addr, name);
		while(current!=null){
			list.push(current);
			current = this.Next(addr, current);
		}

		return keys;
	},

	"GetPairs" : function(addr, name){
       var list = [];
       var current = this.Tail(addr, name);
       while(current!=null){
           var pair = {};
           pair.Key = current;
           pair.Value = this.Main(addr, current);
           list.push(pair);
           current = this.Next(addr, current);
       }       
        return list;
   },
}