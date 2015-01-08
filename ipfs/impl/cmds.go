package impl

import (
	"bytes"
	"errors"
	cmds "github.com/eris-ltd/go-ipfs/commands"
	core "github.com/eris-ltd/go-ipfs/core"
	ccmds "github.com/eris-ltd/go-ipfs/core/commands"
	importer "github.com/eris-ltd/go-ipfs/importer"
	"github.com/eris-ltd/go-ipfs/importer/chunk"
	dag "github.com/eris-ltd/go-ipfs/merkledag"
	pinning "github.com/eris-ltd/go-ipfs/pin"
	ft "github.com/eris-ltd/go-ipfs/unixfs"
	uio "github.com/eris-ltd/go-ipfs/unixfs/io"
	"io"
	"os"
	"path"
	fp "path/filepath"
)

// Much of this fucntionality used to be exported from go-ipfs
// but now its private, so we duplicate.
// Perhaps it is better that way, perhaps not
func cat(node *core.IpfsNode, paths []string) ([]byte, error) {
	readers := make([]io.Reader, 0, len(paths))
	for _, path := range paths {
		dagnode, err := node.Resolver.ResolvePath(path)
		if err != nil {
			return nil, err
		}
		read, err := uio.NewDagReader(dagnode, node.DAG)
		if err != nil {
			return nil, err
		}
		readers = append(readers, read)
	}
	b := new(bytes.Buffer)
	reader := io.MultiReader(readers...)
	io.Copy(b, reader)
	return b.Bytes(), nil
}
func add(n *core.IpfsNode, readers []io.Reader) ([]*dag.Node, error) {
	mp, ok := n.Pinning.(pinning.ManualPinner)
	if !ok {
		return nil, errors.New("invalid pinner type! expected manual pinner")
	}
	dagnodes := make([]*dag.Node, 0)
	for _, reader := range readers {
		node, err := importer.BuildDagFromReader(reader, n.DAG, mp, chunk.DefaultSplitter)
		if err != nil {
			return nil, err
		}
		dagnodes = append(dagnodes, node)
	}
	return dagnodes, nil
}
func addNode(n *core.IpfsNode, node *dag.Node) error {
	err := n.DAG.AddRecursive(node) // add the file to the graph + local storage
	if err != nil {
		return err
	}
	err = n.Pinning.Pin(node, true) // ensure we keep it
	if err != nil {
		return err
	}
	return nil
}
func addFile(n *core.IpfsNode, file cmds.File, added *ccmds.AddOutput) (*dag.Node, error) {
	if file.IsDirectory() {
		return addDir(n, file, added)
	}
	dns, err := add(n, []io.Reader{file})
	if err != nil {
		return nil, err
	}
	//log.Infof("adding file: %s", file.FileName())
	if err := addDagnode(added, file.FileName(), dns[len(dns)-1]); err != nil {
		return nil, err
	}
	return dns[len(dns)-1], nil // last dag node is the file.
}
func addDir(n *core.IpfsNode, dir cmds.File, added *ccmds.AddOutput) (*dag.Node, error) {
	//log.Infof("adding directory: %s", dir.FileName())
	tree := &dag.Node{Data: ft.FolderPBData()}
	for {
		file, err := dir.NextFile()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if file == nil {
			break
		}
		node, err := addFile(n, file, added)
		if err != nil {
			return nil, err
		}
		_, name := path.Split(file.FileName())
		err = tree.AddNodeLink(name, node)
		if err != nil {
			return nil, err
		}
	}
	err := addDagnode(added, dir.FileName(), tree)
	if err != nil {
		return nil, err
	}
	err = addNode(n, tree)
	if err != nil {
		return nil, err
	}
	return tree, nil
}

// addDagnode adds dagnode info to an output object
func addDagnode(output *ccmds.AddOutput, name string, dn *dag.Node) error {
	o, err := getOutput(dn)
	if err != nil {
		return err
	}
	output.Objects = append(output.Objects, o)
	output.Names = append(output.Names, name)
	return nil
}
func getOutput(dagnode *dag.Node) (*ccmds.Object, error) {
	key, err := dagnode.Key()
	if err != nil {
		return nil, err
	}
	output := &ccmds.Object{
		Hash:  key.Pretty(),
		Links: make([]ccmds.Link, len(dagnode.Links)),
	}
	for i, link := range dagnode.Links {
		output.Links[i] = ccmds.Link{
			Name: link.Name,
			Hash: link.Hash.B58String(),
			Size: link.Size,
		}
	}
	return output, nil
}

// recursively get file or directory contents as a cmds.File
func openPath(file *os.File, path string) (cmds.File, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	// for non-directories, return a ReaderFile
	if !stat.IsDir() {
		return &cmds.ReaderFile{path, file}, nil
	}
	// for directories, recursively iterate though children then return as a SliceFile
	contents, err := file.Readdir(0)
	if err != nil {
		return nil, err
	}
	files := make([]cmds.File, 0, len(contents))
	for _, child := range contents {
		childPath := fp.Join(path, child.Name())
		childFile, err := os.Open(childPath)
		if err != nil {
			return nil, err
		}
		f, err := openPath(childFile, childPath)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return &cmds.SliceFile{path, files}, nil
}
