glululemon
==========

Glue packages for wrapping libraries so we can tie blockchains and filesystems (modules) to decervers.

These packages are also used by EPM to facilitate easy plug and play of new blockchains and utilities into the Eris Package Manager.

Note, for a given module to be used with the decerver, it must satisfy the Module interface in `decerver-interfaces/modules` and wrap an object which satisfies the particular interface for the functionality the module provides. For example, `IpfsModule` satisfies both `Module` and `FileSystem`, and wraps an object (`Ipfs`) which satisfies only `FileSystem`. This is necessary so we can bind `Ipfs` to the javascript virtual machine without also exposing the configuration and booting functions in the `IpfsModule` object, but also use the `IpfsModule` as a standalone wrapper for using Ipfs alone or in other programs. One day we will improve this slightly awful state of affairs.

Modules
=========

- blockchaininfo : wraps a blockchain.info API library
- btcd : wraps the bitcoin client written in go. This wrapper is unfinished
- eth : wraps ethereum
- genblock : a simple wrapper on genesis block deployment from thelonious; so you can manage genesis block deployment from epm using a `.pdx` file
- ipfs : wraps the go-ipfs client to provide decentralized file system services
- monk : simple wrapper for the monk module to facilitate easily working with javascript. 
- monkrpc : a basic rpc client for talking to the rpc server in thelonious so you can work with an active chain or run epm deploys to a remote server

Utilities
=========

- utils : basic common utilities like directory creation and paths
- monkutils : utilities for working with thelonious/monk based modules (monk, genblock, monkrpc)
