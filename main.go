package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/cbergoon/merkletree"
)

func main() {
	// for a more elaborate description, see the README
	var factomd = flag.String("factomd", "http://localhost:8088/v2", "the factomd endpoint to use (def: http://localhost:8088/v2)")
	var eckey = flag.String("ec", "", "an entry credit address private key (eg: Es....)")
	var chain = flag.String("chain", "d3bf4593aeeb46fc60b83c0b064e4bf7654a704d8a4583dd4a39bf04f4c35344", "the chain to write snapshot entries to")
	var directory = flag.String("dir", ".", "the path to the directory to snapshot")
	var snaps = flag.String("snaps", ".", "the path to write snapshots to")
	flag.Parse()

	// turn user strings into a format usable by the FAT/factom client
	chainID := factom.NewBytes32(*chain)
	es, err := factom.NewEsAddress(*eckey)
	if err != nil {
		log.Printf("entry credit address not valid: %v", err)
	}

	// initialize a new FAT/factom client
	client := factom.NewClient()
	client.FactomdServer = *factomd

	// retrieve the height of the topmost block
	var heights factom.Heights
	if err := heights.Get(context.Background(), client); err != nil {
		log.Printf("error trying to initialize heights: %v", err)
		return
	}

	// check if the chain exists
	var eblock factom.EBlock
	eblock.ChainID = &chainID
	if _, err := eblock.GetChainHead(context.Background(), client); err != nil {
		log.Printf("chain %s has not been initialized", *chain)
		return
	}

	height := heights.DirectoryBlock
	log.Printf("initialized with height = %d, waiting for next block...", height)

	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		// retrieve the height of the topmost block
		if err := heights.Get(context.Background(), client); err != nil {
			log.Printf("error trying to retrieve heights: %v", err)
		}

		// only act if the block changes
		if heights.DirectoryBlock > height {
			height = heights.DirectoryBlock
			log.Printf("block %d is closed, reached %d", height, height+1)

			// create a snapshot
			snapFile := fmt.Sprintf("snap-%d.log", height+1)
			snapPath := filepath.Join(*snaps, snapFile)
			merkle, err := CreateSnapshot(*directory, snapPath)
			if err != nil {
				log.Printf("unable to create snapshot: %v", err)
				continue
			}

			log.Printf("successfully created snapshot file %s", snapPath)

			// submit a snapshot
			entryHash, err := SubmitSnapshot(client, snapFile, merkle, es, chainID)
			if err != nil {
				log.Printf("unable to submit entry: %v", err)
				continue
			}

			log.Printf("submitted entry for %s: %s", snapFile, entryHash)
		}
	}
}

// CreateSnapshot will create a list of files in a directory, then build a merkle
// tree from the file hashes. The snapshot file is written to disk and the merkle
// root is returned
func CreateSnapshot(read, write string) ([]byte, error) {
	list, err := BuildList(read) // see file.go
	if err != nil {
		return nil, err
	}

	// create a tree from the list of files
	tree, err := merkletree.NewTree([]merkletree.Content(list))
	if err != nil {
		return nil, err
	}

	f, err := os.Create(write)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	// write the list to the snapshot file
	for _, f := range list {
		fi := f.(*FileItem)
		h, _ := f.CalculateHash() // the hash is already cached at this point
		_, err = w.WriteString(fmt.Sprintf("%x %s\n", h, fi.Path))
		if err != nil {
			return nil, err
		}
	}

	if err := w.Flush(); err != nil {
		return nil, err
	}

	return tree.MerkleRoot(), nil
}

// SubmitSnapshot submits the merkle root to the Factom Protocol
func SubmitSnapshot(client *factom.Client, snapFile string, merkle []byte, eckey factom.EsAddress, chain factom.Bytes32) (*factom.Bytes32, error) {
	entry := new(factom.Entry)
	entry.ChainID = &chain
	entry.ExtIDs = append(entry.ExtIDs, []byte(snapFile)) // set the snapshot file name as ExtId[0]
	entry.Content = merkle                                // entries accept up to ~10kb of arbitrary binary data

	// to write an entry, three actions are necessary:
	//  * compose, which creates the Factom Protocol specific binary structure of the entry
	//  * commit, which pays for an entry
	//  * reveal, which sends the data of the entry
	// ComposeCreate performs all three of these steps
	// for more info, see: https://github.com/Factom-Asset-Tokens/factom/blob/481aa2c193455df3416fef056941ebff8a5cac51/entry.go#L171-L181
	if _, err := entry.ComposeCreate(context.Background(), client, eckey); err != nil {
		return &factom.Bytes32{}, err
	}

	return entry.Hash, nil
}
