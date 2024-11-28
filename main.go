package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tomochain/tomochain/common"
	"github.com/tomochain/tomochain/core/types"
	"github.com/tomochain/tomochain/rlp"
)

var (
	headHeaderKey = []byte("LastHeader")
	headBlockKey  = []byte("LastBlock")
	headFastKey   = []byte("LastFast")
	trieSyncKey   = []byte("TrieSync")

	// Data item prefixes (use single byte to avoid mixing data types, avoid `i`).
	headerPrefix        = []byte("h") // headerPrefix + num (uint64 big endian) + hash -> header
	tdSuffix            = []byte("t") // headerPrefix + num (uint64 big endian) + hash + tdSuffix -> td
	numSuffix           = []byte("n") // headerPrefix + num (uint64 big endian) + numSuffix -> hash
	blockHashPrefix     = []byte("H") // blockHashPrefix + hash -> num (uint64 big endian)
	bodyPrefix          = []byte("b") // bodyPrefix + num (uint64 big endian) + hash -> block body
	blockReceiptsPrefix = []byte("r") // blockReceiptsPrefix + num (uint64 big endian) + hash -> block receipts
	lookupPrefix        = []byte("l") // lookupPrefix + hash -> transaction/receipt lookup metadata
	bloomBitsPrefix     = []byte("B") // bloomBitsPrefix + bit (uint16 big endian) + section (uint64 big endian) + hash -> bloom bits

	preimagePrefix    = []byte("secure-key-")      // preimagePrefix + hash -> preimage
	configPrefix      = []byte("ethereum-config-") // config prefix for the db
	BlockchainVersion = []byte("BlockchainVersion")
	// Chain index prefixes (use `i` + single byte to avoid mixing data types).
	BloomBitsIndexPrefix = []byte("iB") // BloomBitsIndexPrefix is the data table of a chain indexer to track its progress

	// used by old db, now only used for conversion
	oldReceiptsPrefix = []byte("receipts-")
	// oldTxMetaSuffix   = []byte{0x01}

	CliqueSnapshotPrefix = []byte("clique-")
	PosvSnapshotPrefix   = []byte("posv-")

	// sectionHeadPrefix = []byte("shead")

	ChtPrefix           = []byte("cht-")
	ChtRootPrefix       = []byte("chtRoot-")
	ChtIndexTablePrefix = []byte("chtIndex-")

	BloomTriePrefix      = []byte("bltRoot-")
	BloomTrieTablePrefix = []byte("blt-")
	BloomTrieIndexPrefix = []byte("bltIndex-")

	ValidSectionCount = []byte("count")
)

func Practice(db *leveldb.DB) {
	logFile, err := os.Create("log/practice.log")
	if err != nil {
		fmt.Println("Error creating log file:", err)
		return
	}
	defer logFile.Close()

	// Create a logger that writes to the file
	logger := func(format string, a ...interface{}) {
		logFile.WriteString(fmt.Sprintf(format, a...))
	}

	blockNumber := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumber, uint64(40))
	logger("Details of Blocknumber:- \nHex: %x \nBytes: %d\n\n\n", blockNumber, blockNumber)

	// create key to get hash (headerPrefix + num (uint64 big endian) + numSuffix)
	hashKey := append(headerPrefix, blockNumber...) // adding prefix
	hashKey = append(hashKey, numSuffix...)         // adding suffix
	logger("Details of leveldb key for Block Hash:- \nType: %T  \nHex: %x \nbytes: %v \nLength:  %d\n\n\n", hashKey, hashKey, hashKey, len(hashKey))

	// Getting hash using hashKey
	blockHash, _ := db.Get(hashKey, nil)
	logger("Details of Block hash:- \nType: %T \nHex: %x \nBytes: %v\n\n\n", blockHash, blockHash, blockHash)

	//Create key to get header (headerPrefix + num (uint64 big endian) + hash)
	headerKey := append(headerPrefix, blockNumber...) // adding prefix
	headerKey = append(headerKey, blockHash...)       // adding suffix
	logger("Details of leveldb key for Block Header:- \nType: %T  \nHex: %x \nVytes: %v \nLength:  %d\n\n\n", headerKey, headerKey, headerKey, len(headerKey))

	//get Block Header data from db
	blockHeaderRLP, _ := db.Get(headerKey, nil)
	logger("Details of Raw Block Header:- \nType: %T  \nHex: %x \nBytes: %v \nLength:  %d\n\n\n", blockHeaderRLP, blockHeaderRLP, blockHeaderRLP, len(blockHeaderRLP))

	blockHeader := new(types.Header)
	rlp.Decode(bytes.NewReader(blockHeaderRLP), blockHeader)
	logger("Details of Header:- \nType: %T  \nHex: %x \nValue: %v\n\n\n", blockHeader, blockHeader, blockHeader)

	bodyKey := append(bodyPrefix, blockNumber...)
	bodyKey = append(bodyKey, blockHash...)
	blockBodyRLP, _ := db.Get(bodyKey, nil)
	logger("Details of Raw Block Body:- \nType: %T  \nHex: %x \nBytes: %v \nLength:  %d\n\n\n", blockBodyRLP, blockBodyRLP, blockBodyRLP, len(blockBodyRLP))

	blockBody := new(types.Body)
	rlp.Decode(bytes.NewReader(blockBodyRLP), blockBody)
	logger("Details of Body:- \nType: %T  \nValue: %v\n\n\n", blockBody, blockBody)

	block := types.NewBlockWithHeader(blockHeader).WithBody(blockBody.Transactions, blockBody.Uncles)
	logger("Block Details:\n%v\n", block)
}

type counter uint64

func (c counter) String() string {
	return fmt.Sprintf("%d", c)
}

func (c counter) Percentage(current uint64) string {
	return fmt.Sprintf("%d", current*100/uint64(c))
}

type stat struct {
	size  common.StorageSize
	count counter
}

func (s *stat) Add(size common.StorageSize) {
	s.size += size
	s.count++
}

func (s *stat) Size() string {
	return s.size.String()
}

func (s *stat) Count() string {
	return s.count.String()
}

func InspectDatabase(db *leveldb.DB) {
	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	var (
		count  int64
		start  = time.Now()
		logged = time.Now()

		// Key-value store statistics
		headers           stat
		bodies            stat
		receipts          stat
		tds               stat
		numHashPairings   stat
		hashNumPairings   stat
		tries             stat
		codes             stat
		txLookups         stat
		accountSnaps      stat
		storageSnaps      stat
		preimages         stat
		bloomBits         stat
		blockchainVersion stat
		posvSnaps         stat
		cliqueSnaps       stat
		ethConfig         stat

		// Ancient store statistics
		ancientHeadersSize  common.StorageSize
		ancientBodiesSize   common.StorageSize
		ancientReceiptsSize common.StorageSize
		ancientTdsSize      common.StorageSize
		ancientHashesSize   common.StorageSize

		// Les statistic
		chtTrieNodes   stat
		bloomTrieNodes stat

		// Meta- and unaccounted data
		metadata stat

		lastHeader stat
		lastBlock  stat
		lastFast   stat
		trieSync   stat

		unaccounted stat

		// Totals
		total common.StorageSize
	)

	inspected, err := os.Create("log/inspect.log")
	if err != nil {
		fmt.Println("Error creating file log:", err)
		return
	}
	defer inspected.Close()

	// Inspect key-value database first.
	for iter.Next() {
		var (
			key  = iter.Key()
			size = common.StorageSize(len(key) + len(iter.Value()))
		)
		total += size
		switch {
		case bytes.HasPrefix(key, headerPrefix) && len(key) == (len(headerPrefix)+8+common.HashLength):
			headers.Add(size)
		case bytes.HasPrefix(key, bodyPrefix) && len(key) == (len(bodyPrefix)+8+common.HashLength):
			bodies.Add(size)
		case bytes.HasPrefix(key, blockReceiptsPrefix) && len(key) == (len(blockReceiptsPrefix)+8+common.HashLength):
			receipts.Add(size)
		case bytes.HasPrefix(key, oldReceiptsPrefix) && len(key) == (len(oldReceiptsPrefix)+common.HashLength):
			receipts.Add(size)
		case bytes.HasPrefix(key, headerPrefix) && bytes.HasSuffix(key, tdSuffix):
			tds.Add(size)
		case bytes.HasPrefix(key, headerPrefix) && bytes.HasSuffix(key, numSuffix):
			numHashPairings.Add(size)
		case bytes.HasPrefix(key, blockHashPrefix) && len(key) == (len(blockHashPrefix)+common.HashLength):
			hashNumPairings.Add(size)
		case len(key) == common.HashLength:
			tries.Add(size)
		case bytes.HasPrefix(key, lookupPrefix) && len(key) == (len(lookupPrefix)+common.HashLength):
			txLookups.Add(size)
		case bytes.HasPrefix(key, preimagePrefix) && len(key) == (len(preimagePrefix)+common.HashLength):
			preimages.Add(size)
		case bytes.HasPrefix(key, bloomBitsPrefix) && len(key) == (len(bloomBitsPrefix)+10+common.HashLength):
			bloomBits.Add(size)
		case bytes.HasPrefix(key, CliqueSnapshotPrefix) && len(key) == 7+common.HashLength:
			cliqueSnaps.Add(size)
		case bytes.HasPrefix(key, PosvSnapshotPrefix) && len(key) == 5+common.HashLength:
			posvSnaps.Add(size)
		case bytes.HasPrefix(key, configPrefix) && len(key) == len(configPrefix)+common.HashLength:
			ethConfig.Add(size)
		case bytes.HasSuffix(key, BlockchainVersion):
			blockchainVersion.Add(size)
		case bytes.HasPrefix(key, ChtRootPrefix) || bytes.HasPrefix(key, ChtIndexTablePrefix) || bytes.HasPrefix(key, ChtPrefix):
			chtTrieNodes.Add(size)
		case bytes.HasPrefix(key, BloomTriePrefix) || bytes.HasPrefix(key, BloomTrieTablePrefix) || bytes.HasPrefix(key, BloomTrieIndexPrefix):
			bloomTrieNodes.Add(size)
		default:
			var accounted bool
			for _, meta := range [][]byte{headHeaderKey, headBlockKey, headFastKey, trieSyncKey} {
				if bytes.Equal(key, meta) {

					if bytes.Equal(meta, headHeaderKey) {
						lastHeader.Add(size)
					} else if bytes.Equal(meta, headBlockKey) {
						lastBlock.Add(size)
					} else if bytes.Equal(meta, headFastKey) {
						lastFast.Add(size)
					} else if bytes.Equal(meta, trieSyncKey) {
						trieSync.Add(size)
					}

					metadata.Add(size)
					accounted = true
					break
				}
			}
			if !accounted {
				unaccounted.Add(size)
			}
		}
		count++
		if count%1000 == 0 && time.Since(logged) > 8*time.Second {
			fmt.Println("Inspecting database", "count", count, "elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
	}

	ancients := counter(0)

	// Display the database statistic.
	stats := [][]string{
		{"Key-Value store", "Headers", headers.Size(), headers.Count()},
		{"Key-Value store", "Bodies", bodies.Size(), bodies.Count()},
		{"Key-Value store", "Receipt lists", receipts.Size(), receipts.Count()},
		{"Key-Value store", "Difficulties", tds.Size(), tds.Count()},
		{"Key-Value store", "Block number->hash", numHashPairings.Size(), numHashPairings.Count()},
		{"Key-Value store", "Block hash->number", hashNumPairings.Size(), hashNumPairings.Count()},
		{"Key-Value store", "Transaction index", txLookups.Size(), txLookups.Count()},
		{"Key-Value store", "Bloombit index", bloomBits.Size(), bloomBits.Count()},
		{"Key-Value store", "Contract codes", codes.Size(), codes.Count()},
		{"Key-Value store", "Trie nodes", tries.Size(), tries.Count()},
		{"Key-Value store", "Trie preimages", preimages.Size(), preimages.Count()},
		{"Key-Value store", "Account snapshot", accountSnaps.Size(), accountSnaps.Count()},
		{"Key-Value store", "Storage snapshot", storageSnaps.Size(), storageSnaps.Count()},
		{"Key-Value store", "Clique snapshots", cliqueSnaps.Size(), cliqueSnaps.Count()},
		{"Key-Value store", "Posv snapshots", posvSnaps.Size(), posvSnaps.Count()},
		{"Key-Value store", "Ethereum config", ethConfig.Size(), ethConfig.Count()},
		{"Key-Value store", "Blockchain Version", blockchainVersion.Size(), blockchainVersion.Count()},
		{"Key-Value store", "Singleton metadata", metadata.Size(), metadata.Count()},
		{"Key-Value store", "Last header", lastHeader.Size(), lastHeader.Count()},
		{"Key-Value store", "Last block", lastBlock.Size(), lastBlock.Count()},
		{"Key-Value store", "Last fast", lastFast.Size(), lastFast.Count()},
		{"Key-Value store", "Trie sync", trieSync.Size(), trieSync.Count()},
		{"Ancient store", "Headers", ancientHeadersSize.String(), ancients.String()},
		{"Ancient store", "Bodies", ancientBodiesSize.String(), ancients.String()},
		{"Ancient store", "Receipt lists", ancientReceiptsSize.String(), ancients.String()},
		{"Ancient store", "Difficulties", ancientTdsSize.String(), ancients.String()},
		{"Ancient store", "Block number->hash", ancientHashesSize.String(), ancients.String()},
		{"Light client", "CHT trie nodes", chtTrieNodes.Size(), chtTrieNodes.Count()},
		{"Light client", "Bloom trie nodes", bloomTrieNodes.Size(), bloomTrieNodes.Count()},
	}
	table := tablewriter.NewWriter(inspected)
	table.SetHeader([]string{"Database", "Category", "Size", "Items"})
	table.SetFooter([]string{"", "Total", total.String(), " "})
	table.AppendBulk(stats)
	table.Render()

	if unaccounted.size > 0 {
		fmt.Println("Database contains unaccounted data", "size", unaccounted.size, "count", unaccounted.count)
	}
}

func main() {
	if err := os.MkdirAll("log", os.ModePerm); err != nil {
		fmt.Println("Error creating log folder:", err)
		return
	}

	db, _ := leveldb.OpenFile("./chaindata", nil)

	Practice(db)
	InspectDatabase(db)
}
