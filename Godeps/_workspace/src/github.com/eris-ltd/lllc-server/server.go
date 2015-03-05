package lllcserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/epm-go/utils"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/go-martini/martini"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"
)

// must have compiler installed!
func homeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

// Server cache location in decerver tree
var ServerCache = path.Join(utils.Lllc, "server")

// Handler for proxy requests (ie. a compile request from langauge other than go)
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	// read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Errorln("err on read http request body", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("body:", string(body))
	req := new(ProxyReq)
	err = json.Unmarshal(body, req)
	if err != nil {
		logger.Errorln("err on read http request body", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var code []byte
	if req.Literal {
		code, err = CompileLiteral(req.Source, req.Language)
	} else {
		code, err = Compile(req.Source)
	}
	resp := NewProxyResponse(code, err)

	respJ, err := json.Marshal(resp)
	if err != nil {
		logger.Errorln("failed to marshal", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(respJ)
}

// Main http request handler
// Read request, compile, build response object, write
func CompileHandler(w http.ResponseWriter, r *http.Request) {
	resp := compileResponse(w, r)
	if resp == nil {
		return
	}
	respJ, err := json.Marshal(resp)
	if err != nil {
		logger.Errorln("failed to marshal", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(respJ)
}

// Convenience wrapper for javascript frontend
func CompileHandlerJs(w http.ResponseWriter, r *http.Request) {
	resp := compileResponse(w, r)
	if resp == nil {
		return
	}
	code := resp.Bytecode
	hexx := hex.EncodeToString(code)
	w.Write([]byte(fmt.Sprintf(`{"bytecode": "%s"}`, hexx)))
}

// read in the files from the request, compile them
func compileResponse(w http.ResponseWriter, r *http.Request) *Response {
	// read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Errorln("err on read http request body", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	// unmarshall body into req struct
	req := new(Request)
	err = json.Unmarshal(body, req)
	if err != nil {
		logger.Errorln("err on json unmarshal of request", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	resp := compileServerCore(req)
	return resp
}

// core compile functionality. used by the server and locally to mimic the server
func compileServerCore(req *Request) *Response {
	var name string
	lang := req.Language
	compiler := Languages[lang]

	c := req.Script
	if c == nil || len(c) == 0 {
		name = "NULLCACHED"
	} else {
		// take sha2 of request object to get tmp filename
		hash := sha256.Sum256(c)
		filename := path.Join(ServerCache, compiler.Ext(hex.EncodeToString(hash[:])))
		name = filename

		// lllc requires a file to read
		// check if filename already exists. if not, write
		if _, err := os.Stat(filename); err != nil {
			ioutil.WriteFile(filename, c, 0644)
		}
	}

	// loop through includes, also save to drive
	for k, v := range req.Includes {
		filename := path.Join(ServerCache, compiler.Ext(k))
		if _, err := os.Stat(filename); err != nil {
			ioutil.WriteFile(filename, v, 0644)
		}
	}
	var resp *Response
	//compile scripts, return bytecode and error
	if name == "NULLCACHED" {

		resp = NewResponse([]byte("NULLCACHED"), nil)
	} else {
		compiled, err := CompileWrapper(name, lang)
		resp = NewResponse(compiled, err)
	}

	return resp
}

// wrapper to cli
func CompileWrapper(filename string, lang string) ([]byte, error) {
	// we need to be in the same dir as the files for sake of includes
	cur, _ := os.Getwd()
	dir := path.Dir(filename)
	dir, _ = filepath.Abs(dir)
	filename = path.Base(filename)

	if _, ok := Languages[lang]; !ok {
		return nil, UnknownLang(lang)
	}

	os.Chdir(dir)
	prgrm, args := Languages[lang].Cmd(filename)
	cmd := exec.Command(prgrm, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		logger.Errorln("Couldn't compile!!", err)
		os.Chdir(cur)
		return nil, err
	}
	os.Chdir(cur)

	outstr := out.String()
	// get rid of new lines at the end
	outstr = strings.TrimSpace(outstr) //"\n")

	b, err := hex.DecodeString(outstr)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Start the compile server
func StartServer(addr string) {
	//martini.Env = martini.Prod
	srv := martini.Classic()
	// Static files
	srv.Use(martini.Static("./web"))

	srv.Post("/compile", CompileHandler)
	srv.Post("/compile2", CompileHandlerJs)

	srv.RunOnAddr(addr)
}

// Start the proxy server
// Dead simple json-rpc so we can compile code from languages other than go
func StartProxy(addr string) {
	srv := martini.Classic()
	srv.Post("/", ProxyHandler)
	srv.RunOnAddr(addr)
}
