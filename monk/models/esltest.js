function test(address){

	sst = esl.single.Value(address,"test1")
	if(sst!="0xDEADBEEF"){
		PrintF("Error when Accessing single type got %s expected 0xdeadbeef",sst)
	}

	dst = esl.double.Value(address,"test2")
	if(dst[0]!="0xdeadbeef" && dst[1]!="0xfeedface"){
		PringF("Error when accessing double type. Got %s %s, expected 0xdeadbeef 0xfeedface",dst[0],dst[1])
	}

	kvt = esl.keyvalue.Value(address,"test2","key1")
	
};