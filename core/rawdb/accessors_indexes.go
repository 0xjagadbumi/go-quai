// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"bytes"
	"math/big"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/core/types"
	"github.com/dominant-strategies/go-quai/ethdb"
	"github.com/dominant-strategies/go-quai/log"
	"google.golang.org/protobuf/proto"
)

// ReadTxLookupEntry retrieves the positional metadata associated with a transaction
// hash to allow retrieving the transaction or receipt by hash.
func ReadTxLookupEntry(db ethdb.Reader, hash common.Hash) *uint64 {
	data, _ := db.Get(txLookupKey(hash))
	if len(data) == 0 {
		return nil
	}
	// Database v6 tx lookup just stores the block number
	if len(data) < common.HashLength {
		number := new(big.Int).SetBytes(data).Uint64()
		return &number
	}
	// Database v4-v5 tx lookup format just stores the hash
	if len(data) == common.HashLength {
		return ReadHeaderNumber(db, common.BytesToHash(data))
	}
	// Finally try database v3 tx lookup format
	protoLegacyTxLookupEntry := new(ProtoLegacyTxLookupEntry)
	err := proto.Unmarshal(data, protoLegacyTxLookupEntry)
	if err != nil {
		db.Logger().WithFields(log.Fields{
			"hash": hash,
			"blob": data,
			"err":  err,
		}).Error("Invalid transaction lookup entry protobuf")
		return nil
	}
	entry := new(LegacyTxLookupEntry)
	entry.ProtoDecode(protoLegacyTxLookupEntry)
	return &entry.BlockIndex
}

// writeTxLookupEntry stores a positional metadata for a transaction,
// enabling hash based transaction and receipt lookups.
func writeTxLookupEntry(db ethdb.KeyValueWriter, hash common.Hash, numberBytes []byte) {
	if err := db.Put(txLookupKey(hash), numberBytes); err != nil {
		db.Logger().WithField("err", err).Fatal("Failed to store transaction lookup entry")
	}
}

// WriteTxLookupEntries is identical to WriteTxLookupEntry, but it works on
// a list of hashes
func WriteTxLookupEntries(db ethdb.KeyValueWriter, number uint64, hashes []common.Hash) {
	numberBytes := new(big.Int).SetUint64(number).Bytes()
	for _, hash := range hashes {
		writeTxLookupEntry(db, hash, numberBytes)
	}
}

// WriteTxLookupEntriesByBlock stores a positional metadata for every transaction from
// a block, enabling hash based transaction and receipt lookups.
func WriteTxLookupEntriesByBlock(db ethdb.KeyValueWriter, wo *types.WorkObject, nodeCtx int) {
	numberBytes := wo.Number(nodeCtx).Bytes()
	for _, tx := range wo.Body().Transactions() {
		writeTxLookupEntry(db, tx.Hash(), numberBytes)
	}
}

// DeleteTxLookupEntry removes all transaction data associated with a hash.
func DeleteTxLookupEntry(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(txLookupKey(hash)); err != nil {
		db.Logger().WithField("err", err).Fatal("Failed to delete transaction lookup entry")
	}
}

// DeleteTxLookupEntries removes all transaction lookups for a given block.
func DeleteTxLookupEntries(db ethdb.KeyValueWriter, hashes []common.Hash) {
	for _, hash := range hashes {
		DeleteTxLookupEntry(db, hash)
	}
}

// ReadTransaction retrieves a specific transaction from the database, along with
// its added positional metadata.
func ReadTransaction(db ethdb.Reader, hash common.Hash) (*types.Transaction, common.Hash, uint64, uint64) {
	blockNumber := ReadTxLookupEntry(db, hash)
	if blockNumber == nil {
		return nil, common.Hash{}, 0, 0
	}
	blockHash := ReadCanonicalHash(db, *blockNumber)
	if blockHash == (common.Hash{}) {
		return nil, common.Hash{}, 0, 0
	}
	wo := ReadWorkObject(db, *blockNumber, blockHash, types.BlockObject)
	if wo == nil {
		db.Logger().WithFields(log.Fields{
			"number": blockNumber,
			"hash":   blockHash,
		}).Error("Transaction referenced missing")
		return nil, common.Hash{}, 0, 0
	}
	for txIndex, tx := range wo.Body().Transactions() {
		if tx.Hash() == hash {
			return tx, blockHash, *blockNumber, uint64(txIndex)
		}
	}
	db.Logger().WithFields(log.Fields{
		"number": *blockNumber,
		"hash":   blockHash,
		"txhash": hash,
	}).Error("Transaction not found")
	return nil, common.Hash{}, 0, 0
}

// ReadBloomBits retrieves the compressed bloom bit vector belonging to the given
// section and bit index from the.
func ReadBloomBits(db ethdb.KeyValueReader, bit uint, section uint64, head common.Hash) ([]byte, error) {
	return db.Get(bloomBitsKey(bit, section, head))
}

// WriteBloomBits stores the compressed bloom bits vector belonging to the given
// section and bit index.
func WriteBloomBits(db ethdb.KeyValueWriter, bit uint, section uint64, head common.Hash, bits []byte) {
	if err := db.Put(bloomBitsKey(bit, section, head), bits); err != nil {
		db.Logger().WithField("err", err).Fatal("Failed to store bloom bits")
	}
}

// DeleteBloombits removes all compressed bloom bits vector belonging to the
// given section range and bit index.
func DeleteBloombits(db ethdb.Database, bit uint, from uint64, to uint64) {
	start, end := bloomBitsKey(bit, from, common.Hash{}), bloomBitsKey(bit, to, common.Hash{})
	it := db.NewIterator(nil, start)
	defer it.Release()

	for it.Next() {
		if bytes.Compare(it.Key(), end) >= 0 {
			break
		}
		if len(it.Key()) != BloomBitsKeyLength {
			continue
		}
		db.Delete(it.Key())
	}
	if it.Error() != nil {
		db.Logger().WithField("err", it.Error()).Fatal("Failed to delete bloom bits")
	}
}
