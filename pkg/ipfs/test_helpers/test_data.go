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

package test_helpers

import (
	"errors"
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/statediff"

	"github.com/vulcanize/vulcanizedb/pkg/ipfs"
)

// AddressToLeafKey hashes an returns an address
func AddressToLeafKey(address common.Address) common.Hash {
	return common.BytesToHash(crypto.Keccak256(address[:]))
}

// Test variables
var (
	BlockNumber     = rand.Int63()
	BlockHash       = "0xfa40fbe2d98d98b3363a778d52f2bcd29d6790b9b3f3cab2b167fd12d3550f73"
	CodeHash        = common.Hex2Bytes("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
	NewNonceValue   = rand.Uint64()
	NewBalanceValue = rand.Int63()
	ContractRoot    = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	StoragePath     = common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes()
	StorageKey      = common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001").Bytes()
	StorageValue    = common.Hex2Bytes("0x03")
	storage         = []statediff.StorageDiff{{
		Key:   StorageKey,
		Value: StorageValue,
		Path:  StoragePath,
		Proof: [][]byte{},
	}}
	emptyStorage           = make([]statediff.StorageDiff, 0)
	address                = common.HexToAddress("0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476592")
	ContractLeafKey        = AddressToLeafKey(address)
	anotherAddress         = common.HexToAddress("0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476593")
	AnotherContractLeafKey = AddressToLeafKey(anotherAddress)
	testAccount            = state.Account{
		Nonce:    NewNonceValue,
		Balance:  big.NewInt(NewBalanceValue),
		Root:     ContractRoot,
		CodeHash: CodeHash,
	}
	valueBytes, _       = rlp.EncodeToBytes(testAccount)
	CreatedAccountDiffs = statediff.AccountDiffsMap{
		ContractLeafKey: {
			Key:     ContractLeafKey.Bytes(),
			Value:   valueBytes,
			Storage: storage,
		},
		AnotherContractLeafKey: {
			Key:     AnotherContractLeafKey.Bytes(),
			Value:   valueBytes,
			Storage: emptyStorage,
		},
	}

	UpdatedAccountDiffs = statediff.AccountDiffsMap{ContractLeafKey: {
		Key:     ContractLeafKey.Bytes(),
		Value:   valueBytes,
		Storage: storage,
	}}

	DeletedAccountDiffs = statediff.AccountDiffsMap{ContractLeafKey: {
		Key:     ContractLeafKey.Bytes(),
		Value:   valueBytes,
		Storage: storage,
	}}

	MockStateDiff = statediff.StateDiff{
		BlockNumber:     BlockNumber,
		BlockHash:       common.HexToHash(BlockHash),
		CreatedAccounts: CreatedAccountDiffs,
		DeletedAccounts: DeletedAccountDiffs,
		UpdatedAccounts: UpdatedAccountDiffs,
	}
	MockStateDiffRlp, _ = rlp.EncodeToBytes(MockStateDiff)

	mockTransaction1 = types.NewTransaction(0, common.HexToAddress("0x0"), big.NewInt(1000), 50, big.NewInt(100), nil)
	mockTransaction2 = types.NewTransaction(1, common.HexToAddress("0x1"), big.NewInt(2000), 100, big.NewInt(200), nil)
	MockTransactions = types.Transactions{mockTransaction1, mockTransaction2}

	mockReceipt1 = types.NewReceipt(common.HexToHash("0x0").Bytes(), false, 50)
	mockReceipt2 = types.NewReceipt(common.HexToHash("0x1").Bytes(), false, 100)
	MockReceipts = types.Receipts{mockReceipt1, mockReceipt2}

	MockHeader = types.Header{
		Time:        0,
		Number:      big.NewInt(1),
		Root:        common.HexToHash("0x0"),
		TxHash:      common.HexToHash("0x0"),
		ReceiptHash: common.HexToHash("0x0"),
	}
	MockBlock       = types.NewBlock(&MockHeader, MockTransactions, nil, MockReceipts)
	MockBlockRlp, _ = rlp.EncodeToBytes(MockBlock)

	MockStatediffPayload = statediff.Payload{
		BlockRlp:     MockBlockRlp,
		StateDiffRlp: MockStateDiffRlp,
		Err:          nil,
	}

	EmptyStatediffPayload = statediff.Payload{
		BlockRlp:     []byte{},
		StateDiffRlp: []byte{},
		Err:          nil,
	}

	ErrStatediffPayload = statediff.Payload{
		BlockRlp:     []byte{},
		StateDiffRlp: []byte{},
		Err:          errors.New("mock error"),
	}

	MockIPLDPayload = ipfs.IPLDPayload{}

	MockCIDPayload = ipfs.CIDPayload{
		BlockNumber: "1",
		BlockHash:   "0x0",
		HeaderCID:   "mockHeaderCID",
		TransactionCIDs: map[common.Hash]*ipfs.TrxMetaData{
			common.HexToHash("0x0"): {
				CID:  "mockTrxCID1",
				To:   "mockTo1",
				From: "mockFrom1",
			},
			common.HexToHash("0x1"): {
				CID:  "mockTrxCID2",
				To:   "mockTo2",
				From: "mockFrom2",
			},
		},
		ReceiptCIDs: map[common.Hash]*ipfs.ReceiptMetaData{
			common.HexToHash("0x0"): {
				CID:     "mockReceiptCID1",
				Topic0s: []string{"mockTopic1"},
			},
			common.HexToHash("0x1"): {
				CID:     "mockReceiptCID2",
				Topic0s: []string{"mockTopic1", "mockTopic2"},
			},
		},
		StateLeafCIDs: map[common.Hash]string{
			common.HexToHash("0x0"): "mockStateCID1",
			common.HexToHash("0x1"): "mockStateCID2",
		},
		StorageLeafCIDs: map[common.Hash]map[common.Hash]string{
			common.HexToHash("0x0"): {
				common.HexToHash("0x0"): "mockStorageCID1",
			},
			common.HexToHash("0x1"): {
				common.HexToHash("0x1"): "mockStorageCID2",
			},
		},
	}
)