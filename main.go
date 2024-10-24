package main

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tomochain/tomochain/core/types"
	"github.com/tomochain/tomochain/rlp"
)

var (
	// headHeaderKey = []byte("LastHeader")
	// headBlockKey  = []byte("LastBlock")
	// headFastKey   = []byte("LastFast")
	// trieSyncKey   = []byte("TrieSync")

	// Data item prefixes (use single byte to avoid mixing data types, avoid `i`).
	headerPrefix = []byte("h") // headerPrefix + num (uint64 big endian) + hash -> header
	// tdSuffix            = []byte("t") // headerPrefix + num (uint64 big endian) + hash + tdSuffix -> td
	numSuffix = []byte("n") // headerPrefix + num (uint64 big endian) + numSuffix -> hash
	// blockHashPrefix     = []byte("H") // blockHashPrefix + hash -> num (uint64 big endian)
	bodyPrefix = []byte("b") // bodyPrefix + num (uint64 big endian) + hash -> block body
	// blockReceiptsPrefix = []byte("r") // blockReceiptsPrefix + num (uint64 big endian) + hash -> block receipts
	// lookupPrefix        = []byte("l") // lookupPrefix + hash -> transaction/receipt lookup metadata
	// bloomBitsPrefix     = []byte("B") // bloomBitsPrefix + bit (uint16 big endian) + section (uint64 big endian) + hash -> bloom bits

	// preimagePrefix = "secure-key-"              // preimagePrefix + hash -> preimage
	// configPrefix   = []byte("ethereum-config-") // config prefix for the db

	// Chain index prefixes (use `i` + single byte to avoid mixing data types).
	BloomBitsIndexPrefix = []byte("iB") // BloomBitsIndexPrefix is the data table of a chain indexer to track its progress

	// used by old db, now only used for conversion
	// oldReceiptsPrefix = []byte("receipts-")
	// oldTxMetaSuffix   = []byte{0x01}

	// ErrChainConfigNotFound = errors.New("ChainConfig not found") // general config not found error

	// preimageCounter    = metrics.NewRegisteredCounter("db/preimage/total", nil)
	// preimageHitCounter = metrics.NewRegisteredCounter("db/preimage/hits", nil)
)

func main() {
	// Connection to leveldb
	db, _ := leveldb.OpenFile("./chaindata", nil)

	// 40 to bytes (Big endian)
	blockNumber := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumber, uint64(40))
	fmt.Printf("Details of Blocknumber:- \nHex: %x \nBytes: %d\n\n\n", blockNumber, blockNumber)

	// create key to get hash (headerPrefix + num (uint64 big endian) + numSuffix)
	hashKey := append(headerPrefix, blockNumber...) // adding prefix
	hashKey = append(hashKey, numSuffix...)         // adding suffix
	fmt.Printf("Details of leveldb key for Block Hash:- \nType: %T  \nHex: %x \nbytes: %v \nLength:  %d\n\n\n", hashKey, hashKey, hashKey, len(hashKey))

	// Getting hash using hashKey
	blockHash, _ := db.Get(hashKey, nil)
	fmt.Printf("Details of Block hash:- \nType: %T \nHex: %x \nBytes: %v\n\n\n", blockHash, blockHash, blockHash)

	//Create key to get header (headerPrefix + num (uint64 big endian) + hash)
	headerKey := append(headerPrefix, blockNumber...) // adding prefix
	headerKey = append(headerKey, blockHash...)       // adding suffix
	fmt.Printf("Details of leveldb key for Block Header:- \nType: %T  \nHex: %x \nVytes: %v \nLength:  %d\n\n\n", headerKey, headerKey, headerKey, len(headerKey))

	//get Block Header data from db
	blockHeaderRLP, _ := db.Get(headerKey, nil)
	fmt.Printf("Details of Raw Block Header:- \nType: %T  \nHex: %x \nBytes: %v \nLength:  %d\n\n\n", blockHeaderRLP, blockHeaderRLP, blockHeaderRLP, len(blockHeaderRLP))

	blockHeader := new(types.Header)
	rlp.Decode(bytes.NewReader(blockHeaderRLP), blockHeader)
	fmt.Printf("Details of Header:- \nType: %T  \nHex: %x \nValue: %v\n\n\n", blockHeader, blockHeader, blockHeader)

	bodyKey := append(bodyPrefix, blockNumber...)
	bodyKey = append(bodyKey, blockHash...)
	blockBodyRLP, _ := db.Get(bodyKey, nil)
	fmt.Printf("Details of Raw Block Body:- \nType: %T  \nHex: %x \nBytes: %v \nLength:  %d\n\n\n", blockBodyRLP, blockBodyRLP, blockBodyRLP, len(blockBodyRLP))

	blockBody := new(types.Body)
	rlp.Decode(bytes.NewReader(blockBodyRLP), blockBody)
	fmt.Printf("Details of Body:- \nType: %T  \nValue: %v\n\n\n", blockBody, blockBody)

	block := types.NewBlockWithHeader(blockHeader).WithBody(blockBody.Transactions, blockBody.Uncles)
	fmt.Println(block)
}
