package impl

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/eris-ltd/decerver-interfaces/modules"
	proto "github.com/eris-ltd/go-ipfs/Godeps/_workspace/src/code.google.com/p/goprotobuf/proto"
	mh "github.com/eris-ltd/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-multihash"
	"github.com/eris-ltd/go-ipfs/blocks"
	mdag "github.com/eris-ltd/go-ipfs/merkledag"
	uio "github.com/eris-ltd/go-ipfs/unixfs/io"
	ftpb "github.com/eris-ltd/go-ipfs/unixfs/pb"
	"github.com/eris-ltd/go-ipfs/util"
	"github.com/eris-ltd/go-ipfs/Godeps/_workspace/src/code.google.com/p/go.net/context"
	"github.com/eris-ltd/go-ipfs/commands"
	"github.com/eris-ltd/go-ipfs/config"
	"github.com/eris-ltd/go-ipfs/core"
	cmds "github.com/eris-ltd/go-ipfs/core/commands"
	"io"
	"os"
	"strings"
	"time"
)

var StreamSize = 1024

type Ipfs struct {
	node *core.IpfsNode
	cfg  *config.Config
}

// NOTE: Init is in the init file

func (ipfs *Ipfs) Start() error {
	ctx := context.Background()
	n, err := core.NewIpfsNode(ctx, ipfs.cfg, true)
	if err != nil {
		return err
	}
	ipfs.node = n

	return nil
}

// TODO: UDP socket won't close
// https://github.com/jbenet/go-ipfs/issues/389
func (ipfs *Ipfs) Shutdown() error {
	// TODO close
	if n := ipfs.node.Network; n != nil {
		n.Close()
	}
	return nil
}

// ethereum stores hashes as 32 bytes, but ipfs expects base58 encoding
// thus our convention is that params can be a path, but it must have only a single leading hash (hex encoded)
// and it must lead with it
// TODO: purpose this...
func (ipfs *Ipfs) Get(cmd string, params ...string) (interface{}, error) {
	// ipfs
	/*
	   n := ipfs.node
	   if len(params) == 0{
	   return ipfs.getCmd(cmd)
	   }*/
	return nil, errors.New("Invalid commmand")
}
func (ipfs *Ipfs) GetObject(hash string) (interface{}, error) {
	// return raw file bytes or a dir tree
	fpath, err := hexPath2B58(hash)
	if err != nil {
		return nil, err
	}
	nd, err := ipfs.node.Resolver.ResolvePath(fpath)
	if err != nil {
		return nil, err
	}
	pb := new(ftpb.Data)
	err = proto.Unmarshal(nd.Data, pb)
	if err != nil {
		return nil, err
	}
	if pb.GetType() == ftpb.Data_Directory {
		return ipfs.GetTree(hash, -1)
	} else {
		return ipfs.GetFile(hash)
	}
}
func (ipfs *Ipfs) GetBlock(hash string) ([]byte, error) {
	h, err := hex.DecodeString(hash)
	if err != nil {
		return nil, err
	}
	k := util.Key(h)
	ctx, _ := context.WithTimeout(context.TODO(), time.Second*5)
	fmt.Printf("IPFS STUFF: node: %v\n", ipfs.node)
	fmt.Printf("IPFS STUFF: Blocks: %v\n", ipfs.node.Blocks)
	b, err := ipfs.node.Blocks.GetBlock(ctx, k)
	if err != nil {
		return nil, fmt.Errorf("block get: %v", err)
	}
	return b.Data, nil
}
func (ipfs *Ipfs) GetFile(hash string) ([]byte, error) {
	h, err := hexPath2B58(hash)
	if err != nil {
		return nil, err
	}
	// buf := bytes.NewBuffer(nil)
	b, err := cat(ipfs.node, []string{h}) //cmds.Cat(ipfs.node, []string{h}, nil, buf)
	if err != nil {
		return nil, err
	}
	return b, nil
}
func (ipfs *Ipfs) GetStream(hash string) (chan []byte, error) {
	fpath, err := hexPath2B58(hash)
	if err != nil {
		return nil, err
	}
	dagnode, err := ipfs.node.Resolver.ResolvePath(fpath)
	if err != nil {
		return nil, fmt.Errorf("catFile error: %v", err)
	}
	read, err := uio.NewDagReader(dagnode, ipfs.node.DAG)
	if err != nil {
		return nil, fmt.Errorf("cat error: %v", err)
	}
	ch := make(chan []byte)
	var n int
	go func() {
		for err != io.EOF {
			b := make([]byte, StreamSize)
			// read from reader 1024 bytes at a time
			n, err = read.Read(b)
			if err != nil && err != io.EOF {
				//return nil, err
				break
				// how to handle these errors?!
			}
			// broadcast on channel
			ch <- b[:n]
		}
		close(ch)
	}()
	return ch, nil
}

// TODO: depth
func (ipfs *Ipfs) GetTree(hash string, depth int) (modules.JsObject, error) {
	fpath, err := hexPath2B58(hash)
	if err != nil {
		return nil, err
	}
	nd, err1 := ipfs.node.Resolver.ResolvePath(fpath)
	if err1 != nil {
		return nil, err1
	}
	mhash, err2 := nd.Multihash()
	if err2 != nil {
		return nil, err2
	}
	tree := getTreeNode("", hex.EncodeToString(mhash))
	err3 := grabRefs(ipfs.node, nd, tree)
	return tree, err3
}
func (ipfs *Ipfs) getCmd(cmd string) (interface{}, error) {
	return nil, nil
}

// ...
func (ipfs *Ipfs) Push(cmd string, params ...string) (string, error) {
	return "", errors.New("Invalid cmd")
}
func (ipfs *Ipfs) PushBlock(data []byte) (string, error) {
	b := blocks.NewBlock(data)
	k, err := ipfs.node.Blocks.AddBlock(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString([]byte(k)), nil
}
func (ipfs *Ipfs) PushBlockString(data string) (string, error) {
	return ipfs.PushBlock([]byte(data))
}
func (ipfs *Ipfs) PushFile(fpath string) (string, error) {
	file, err := os.Open(fpath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	f := &commands.ReaderFile{
		Filename: fpath,
		Reader:   file,
	}
	added := &cmds.AddOutput{}
	nd, err := addFile(ipfs.node, f, added)
	if err != nil {
		return "", err
	}
	h, err := nd.Multihash()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h), nil
	//return ipfs.PushTree(fpath, 1)
}
func (ipfs *Ipfs) PushTree(fpath string, depth int) (string, error) {
	ff, err := os.Open(fpath)
	if err != nil {
		return "", err
	}
	f, err := openPath(ff, fpath)
	if err != nil {
		return "", err
	}
	added := &cmds.AddOutput{}
	nd, err := addDir(ipfs.node, f, added)
	if err != nil {
		return "", err
	}
	h, err := nd.Multihash()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h), nil
}

// Key manager functions.
// Note in ipfs (in contrast with a blockchain), one is much less likely
// to change keys, as there are accrued benefits to sticking with a single key,
// and there is no notion of "transactions"
// An ipfs ID is simply the multihash of the publickey
func (ipfs *Ipfs) ActiveAddress() string {
	return hex.EncodeToString(ipfs.node.Identity.ID())
}

// Ipfs node's only have one address
func (ipfs *Ipfs) Address(n int) (string, error) {
	return ipfs.ActiveAddress(), nil
}
func (ipfs *Ipfs) SetAddress(addr string) error {
	return fmt.Errorf("It is not possible to set the ipfs node address without restarting.")
}
func (ipfs *Ipfs) SetAddressN(n int) error {
	return fmt.Errorf("It is not possible to set the ipfs node address without restarting.")
}

// We don't create new addresses on the fly
func (ipfs *Ipfs) NewAddress(set bool) (string, error) {
	return "", fmt.Errorf("It is not possible to create new addresses during runtime.")
}

// we only have one ipfs address
func (ipfs *Ipfs) AddressCount() int {
	return 1
}
func HexToB58(s string) (string, error) {
	var b []byte
	if len(s) > 2 {
		if s[:2] == "0x" {
			s = s[2:]
		}
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return "", err
	}
	bmh := mh.Multihash(b)
	return bmh.B58String(), nil //b58.Encode(b), nil
}

// should this return 0x prefixed?
func B58ToHex(s string) (string, error) {
	r, err := mh.FromB58String(s) //b58.Decode(s) with panic recovery
	if err != nil {
		return "", err
	}
	h := hex.EncodeToString(r)
	return "0x" + h, nil
}

// convert path beginning with 32 byte hex string to path beginning with base58 encoded
func hexPath2B58(p string) (string, error) {
	var err error
	p = strings.TrimLeft(p, "/") // trim leading slash
	spl := strings.Split(p, "/") // split path
	leadingHash := spl[0]
	spl[0], err = HexToB58(leadingHash) // convert leading hash to base58
	if err != nil {
		return "", err
	}
	if len(spl) > 1 {
		return strings.Join(spl, "/"), nil
	}
	return spl[0], nil
}
func getTreeNode(name, hash string) modules.JsObject {
	obj := make(modules.JsObject)
	obj["Nodes"] = make([]modules.JsObject, 0)
	obj["Name"] = name
	obj["Hash"] = hash
	return obj
}
func grabRefs(n *core.IpfsNode, nd *mdag.Node, tree modules.JsObject) error {
	for _, link := range nd.Links {
		h := link.Hash
		newNode := getTreeNode(link.Name, h.B58String())
		nd, err := n.DAG.Get(util.Key(h))
		if err != nil {
			//log.Errorf("error: cannot retrieve %s (%s)", h.B58String(), err)
			return err
		}
		err = grabRefs(n, nd, newNode)
		if err != nil {
			return err
		}
		nds := tree["Nodes"].([]modules.JsObject)
		nds = append(nds, newNode)
	}
	return nil
}
