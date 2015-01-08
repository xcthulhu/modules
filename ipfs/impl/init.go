package impl

import (
	"encoding/base64"
	"fmt"
	"github.com/eris-ltd/go-ipfs/config"
	"github.com/eris-ltd/go-ipfs/crypto"
	"github.com/eris-ltd/go-ipfs/peer"
	"github.com/eris-ltd/go-ipfs/repo"
	"github.com/eris-ltd/go-ipfs/util"
	"github.com/eris-ltd/go-ipfs/util/debugerror"
	"os"
	"os/user"
	"path"
	"path/filepath"
)

// Keep this higher then 1024
const nBitsForKeypairDefault = 4096

func (ipfs *Ipfs) Init() error {
	fmt.Println("IPFS: initializing")
	
	// TODO have had minor issues with path dependent config. Will
	// fix. Sticking to ~/.go-ipfs for now
	usr, uErr := user.Current()
	
	if uErr != nil {
		return uErr
	}
	
	rootDir := path.Join(usr.HomeDir, ".go-ipfs")
	cfg, err := config.Load(path.Join(rootDir,"config"))

	if err != nil {
		fmt.Println("IPFS config does not exist, creating... (this may take a few seconds)")
		cfName, cErr := config.Filename(rootDir)
		if cErr != nil {
			return cErr
		}
		cfg, err = initConfig(cfName)
		if err != nil {
			return err
		}
	}
	
	// Make this the active configuration file.
	ipfs.cfg = cfg
	// TODO add settings later.
	util.SetLogLevel("*", "debug")
	fmt.Println("IPFS: init done")
	
	return nil
}

func initConfig(configFilename string) (*config.Config, error) {
	// TODO No overriding atm.
	ds, err := datastoreConfig("")
	if err != nil {
		return nil, err
	}

	identity, err := identityConfig(nBitsForKeypairDefault)
	if err != nil {
		return nil, err
	}

	logConfig, err := initLogs("") // TODO allow user to override dir
	if err != nil {
		return nil, err
	}

	conf := &config.Config{

		// setup the node's default addresses.
		// Note: two swarm listen addrs, one tcp, one utp.
		Addresses: config.Addresses{
			Swarm: []string{
				"/ip4/0.0.0.0/tcp/4001",
				"/ip4/0.0.0.0/udp/4002/utp",
			},
			API: "/ip4/127.0.0.1/tcp/5001",
		},

		Bootstrap: []*config.BootstrapPeer{
			&config.BootstrapPeer{ // Use these hardcoded bootstrap peers for now.
				// mars.i.ipfs.io
				PeerID:  "QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
				Address: "/ip4/104.131.131.82/tcp/4001",
			},
			&config.BootstrapPeer{
				// Neptune
				PeerID:  "QmSoLnSGccFuZQJzRadHn95W2CrSFmZuTdDWP8HXaHca9z",
				Address: "/ip4/104.236.176.52/tcp/4001",
			},
			&config.BootstrapPeer{
				// Pluto
				PeerID:  "QmSoLpPVmHKQ4XTPdz8tjDFgdeRFkpV8JgYq8JVJ69RrZm",
				Address: "/ip4/104.236.179.241/tcp/4001",
			},
			&config.BootstrapPeer{
				// Uranus
				PeerID:  "QmSoLueR4xBeUbY9WZ9xGUUxunbKWcrNFTDAadQJmocnWm",
				Address: "/ip4/162.243.248.213/tcp/4001",
			},
			&config.BootstrapPeer{
				// Saturn
				PeerID:  "QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
				Address: "/ip4/128.199.219.111/tcp/4001",
			},
		},

		Datastore: ds,

		Logs: logConfig,

		Identity: identity,

		// setup the node mount points.
		Mounts: config.Mounts{
			IPFS: "/ipfs",
			IPNS: "/ipns",
		},

		// tracking ipfs version used to generate the init folder and adding
		// update checker default setting.
		Version: config.VersionDefaultValue(),
	}

	if err := config.WriteConfigFile(configFilename, conf); err != nil {
		return nil, err
	}

	return conf, nil
}

// identityConfig initializes a new identity.
func identityConfig(nbits int) (config.Identity, error) {
	// TODO guard higher up
	ident := config.Identity{}
	if nbits < 1024 {
		return ident, debugerror.New("Bitsize less than 1024 is considered unsafe.")
	}

	fmt.Printf("generating key pair...")
	sk, pk, err := crypto.GenerateKeyPair(crypto.RSA, nbits)
	if err != nil {
		return ident, err
	}
	fmt.Printf("done\n")

	// currently storing key unencrypted. in the future we need to encrypt it.
	// TODO(security)
	skbytes, err := sk.Bytes()
	if err != nil {
		return ident, err
	}
	ident.PrivKey = base64.StdEncoding.EncodeToString(skbytes)

	id, err := peer.IDFromPubKey(pk)
	if err != nil {
		return ident, err
	}
	ident.PeerID = id.Pretty()
	fmt.Printf("peer identity: %s\n", ident.PeerID)
	return ident, nil
}

// initLogs initializes the event logger at the specified path. It uses the
// default log path if no path is provided.
func initLogs(logpath string) (config.Logs, error) {
	if len(logpath) == 0 {
		var err error
		logpath, err = config.LogsPath("")
		if err != nil {
			return config.Logs{}, debugerror.Wrap(err)
		}
	}
	err := initCheckDir(logpath)
	if err != nil {
		return config.Logs{}, debugerror.Errorf("logs: %s", err)
	}
	conf := config.Logs{
		Filename: path.Join(logpath, "events.log"),
	}
	err = repo.ConfigureEventLogger(conf)
	if err != nil {
		return conf, err
	}
	return conf, nil
}

// initCheckDir ensures the directory exists and is writable
func initCheckDir(path string) error {
	// Construct the path if missing
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	// Check the directory is writeable
	if f, err := os.Create(filepath.Join(path, "._check_writeable")); err == nil {
		os.Remove(f.Name())
	} else {
		return debugerror.New("'" + path + "' is not writeable")
	}
	return nil
}

func datastoreConfig(dspath string) (config.Datastore, error) {
	ds := config.Datastore{}
	if len(dspath) == 0 {
		var err error
		dspath, err = config.DataStorePath("")
		if err != nil {
			return ds, err
		}
	}
	ds.Path = dspath
	ds.Type = "leveldb"

	err := initCheckDir(dspath)
	if err != nil {
		return ds, debugerror.Errorf("datastore: %s", err)
	}

	return ds, nil
}
