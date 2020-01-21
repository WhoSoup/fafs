# Factomizing a File System

This app showcases how one could use the Factom Protocol in order to timestamp the contents of a directory, securing both the contents of the files and the point in time they existed. It monitors
a Factom node every minute until a new block height has been reached, at which point a snapshot of the given directory is created and its merkle root submitted. *While the files are read, no file data is written to the blockchain.*

Snapshots are files containing the file hashes and paths of every file in the directory and its subdirectories. For the purposes of demonstration, the snapshot files are stored on disk, however they can be deterministically recreated as long as the contents of the directory do not change. In lieu of creating the merkle root of a directory in this manner, a directory's [IPFS CID](https://docs.ipfs.io/guides/concepts/cid/) could also be used.

## Installation

FAFS uses golang 1.13 but otherwise requires no special dependencies outside of go mod.

```
go get github.com/WhoSoup/fafs
go install github.com/WhoSoup/fafs
``` 

## Running

The demo requires several parameters in order to function:

* `-factomd`: The location of a Factom node API endpoint. The default setting (`http://localhost:8088/v2`) works with a local node. If you do not want to run your own node, you can specify the [Open Node](https://api.factomd.net/) via `-factomd=https://api.factomd.net/v2`
* `-ec`: The private key of a Factom Entry Credit address (`Fs...`). The address needs to be funded with a sufficient amount of entry credits, using approximately 144 a day
* `-chain`: The Factom chain that entries should be written to. The default value (`d3bf4593aeeb46fc60b83c0b064e4bf7654a704d8a4583dd4a39bf04f4c35344`) is the result of the chain with the name `FAFS` + `Example` + `Chain`
* `-dir`: The directory to make snapshots of. Includes all files and subdirectories
* `-snaps`: The directory to save snapshot files to. Size depends on the amount of files in the monitoring directory. Makes approximately one file every 10 minutes, named `snap-###.log`, where `###` is the block height it was submitted to

## Adapting for Production

As it stands, the app only serves as a showcase and the corresponding management of directory snapshots and backups is not addressed. A more realistic scenario would be to monitor a backup system that takes immutable snapshots of websites or database backups. Every ten minutes (or every hour, or once a day) a Factom snapshot is created for all new items in the backup system. Items that have already been Factomized once do not need to be included in a second snapshot.

Another feature to add is the cryptographic signing of entries. Since everyone is free to write to every chain, there is no way to distinguish legitimate entries from fakes. For purposes of proving *existence*, it is not necessary to do so, but there are some cases where it is desired to filter out all unnecessary entries. This can be accomplished, for example, by signing (`Height` + `ChainID` + `ExtId[0]` + `Content`) and storing the signature in `ExtID[1]`