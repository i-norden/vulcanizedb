// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package ipfs

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

// IPLDPayload is a custom type which packages ETH data for the IPFS publisher
type IPLDPayload struct {
	HeaderRLP       []byte
	BlockNumber     *big.Int
	BlockHash       common.Hash
	BlockBody       *types.Body
	TrxMetaData     []*TrxMetaData
	Receipts        types.Receipts
	ReceiptMetaData []*ReceiptMetaData
	StateLeafs      map[common.Hash][]byte
	StorageLeafs    map[common.Hash]map[common.Hash][]byte
}

// CIDPayload is a struct to hold all the CIDs and their meta data
type CIDPayload struct {
	BlockNumber     string
	BlockHash       string
	HeaderCID       string
	UncleCIDS       map[common.Hash]string
	TransactionCIDs map[common.Hash]*TrxMetaData
	ReceiptCIDs     map[common.Hash]*ReceiptMetaData
	StateLeafCIDs   map[common.Hash]string
	StorageLeafCIDs map[common.Hash]map[common.Hash]string
}

// ReceiptMetaData wraps some additional data around our receipt CIDs for indexing
type ReceiptMetaData struct {
	CID     string
	Topic0s []string
}

// TrxMetaData wraps some additional data around our transaction CID for indexing
type TrxMetaData struct {
	CID  string
	To   string
	From string
}