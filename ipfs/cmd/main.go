package main

import (
	"fmt"
	"log"
	"time"
	//"encoding/hex"
	"github.com/eris-ltd/modules/ipfs"
	"github.com/eris-ltd/decerver-interfaces/modules"
)

// This guy is used for some manual testing / debugging on a live network
// and its contents at any given time are relatively unimportant.
// One day we write better infrastructure for testing this stuff

func main() {
	i := ipfs.NewIpfs()
	err := i.Init()
	if err != nil {
		log.Fatal(err)
	}
	start := time.Now()
	err = i.Start()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("startup took:", time.Since(start))

	//c := "QmVHdqmE5x55kZaavWUmscLmieusDdZhQBP5mjZHwMB3U9"
	//c := "Qmb8zwr341xu5uUWwxvVKbZs1ZbjJRJJ965tnV9HDeVUkH"
	c := "QmVq6uMzsKg7x5mDEyLS5p5xiyTQ49LR8kFk1wnFDhodzz"
	h, _ := ipfs.B58ToHex(c)
	/*
	   a, _ := i.Get("file", h)
	   fmt.Println(string(a.([]byte)))

	   g, _ := i.Get("tree", ipfs.B58ToHex("QmaKxiCScMY6BG1eq228F2fDJmjxZ53MJ8MtEyEJZr3v44"))
	   t := g.(modules.FsNode)
	   printTree(&t)
	*/
	/*
	   ch, _ := i.Get("stream", h)
	   for r := range ch.(chan []byte){
	       fmt.Println(string(r))
	   }

	   a := hex.EncodeToString([]byte("fuck you"))
	   fmt.Println("#####")
	   k, _ := i.Push("block", a)
	   fmt.Println(k)
	   aa, err := i.Get("block", k)
	   if err != nil{
	       fmt.Println(err)
	   }
	   fmt.Println(string(aa.([]byte)))
	*/
	time.Sleep(time.Second * 5)
	fmt.Println("calling get file...")
	j := i.GetFile(h)
	a := j["Data"]
	e := j["Err"]
	fmt.Println(a.(string), e)
	i.Shutdown()
}

func printTree(t *modules.FsNode) {
	fmt.Println(t.Name)
	for _, tt := range t.Nodes {
		printTree(tt)
	}
}
