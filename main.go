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
	var factomd = flag.String("factomd", "http://localhost:8088", "the factomd endpoint to use (def: http://localhost:8088)")
	var eckey = flag.String("ec", "", "an entry credit address private key (eg: Es....)")
	// 4cb8ea4d504852de583ac149b3c28b88c6ee550fadb122e13d824214218d7809 = ffs + testchain
	var chain = flag.String("chain", "4cb8ea4d504852de583ac149b3c28b88c6ee550fadb122e13d824214218d7809", "the chain to write snapshot entries to")
	var directory = flag.String("dir", ".", "the path to the directory to snapshot")
	var snaps = flag.String("snaps", ".", "the path to write snapshots to")
	flag.Parse()

	es, err := factom.NewEsAddress(*eckey)
	if err != nil {
		log.Printf("entry credit address not valid: %v", err)
	}

	client := factom.NewClient()
	client.FactomdServer = *factomd

	var heights factom.Heights
	if err := heights.Get(context.Background(), client); err != nil {
		log.Printf("error trying to initialize heights: %v", err)
		return
	}

	chainID := factom.NewBytes32(*chain)

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
		if err := heights.Get(context.Background(), client); err != nil {
			log.Printf("error trying to retrieve heights: %v", err)
		}

		if heights.DirectoryBlock > height {
			height = heights.DirectoryBlock
			log.Printf("reached height = %d", height)

			snapFile := fmt.Sprintf("snap-%d.log", height)
			snapPath := filepath.Join(*snaps, snapFile)

			merkle, err := CreateSnapshot(*directory, snapPath)
			if err != nil {
				log.Printf("unable to create snapshot: %v", err)
				continue
			}

			log.Printf("successfully created snapshot file %s", snapPath)

			entryHash, err := SubmitSnapshot(client, snapFile, merkle, es, chainID)
			if err != nil {
				log.Printf("unable to submit entry: %v", err)
				continue
			}

			log.Printf("submitted entry for %s: %x", snapFile, entryHash)
		}
	}
}

func CreateSnapshot(read, write string) ([]byte, error) {
	list, err := BuildList(read)
	if err != nil {
		return nil, err
	}

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

	for _, f := range list {
		fi := f.(FileItem)
		h, _ := f.CalculateHash()
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

func SubmitSnapshot(client *factom.Client, snapFile string, merkle []byte, eckey factom.EsAddress, chain factom.Bytes32) (factom.Bytes32, error) {
	entry := new(factom.Entry)
	entry.ChainID = &chain
	entry.ExtIDs = append(entry.ExtIDs, []byte(snapFile))
	entry.Content = merkle

	return entry.ComposeCreate(context.Background(), client, eckey)
}
