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
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ipfs/go-block-format"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	log "github.com/sirupsen/logrus"
)

// IPLDFetcher is an interface for fetching IPLDs
type IPLDFetcher interface {
	FetchIPLDs(cids CIDWrapper) (*IPLDWrapper, error)
}

// EthIPLDFetcher is used to fetch ETH IPLD objects from IPFS
type EthIPLDFetcher struct {
	BlockService blockservice.BlockService
}

// NewIPLDFetcher creates a pointer to a new IPLDFetcher
func NewIPLDFetcher(ipfsPath string) (*EthIPLDFetcher, error) {
	blockService, err := InitIPFSBlockService(ipfsPath)
	if err != nil {
		return nil, err
	}
	return &EthIPLDFetcher{
		BlockService: blockService,
	}, nil
}

// FetchIPLDs is the exported method for fetching and returning all the IPLDS specified in the CIDWrapper
func (f *EthIPLDFetcher) FetchIPLDs(cids CIDWrapper) (*IPLDWrapper, error) {

	log.Debug("fetching iplds")
	blocks := &IPLDWrapper{
		BlockNumber:  cids.BlockNumber,
		Headers:      make([]blocks.Block, 0),
		Uncles:       make([]blocks.Block, 0),
		Transactions: make([]blocks.Block, 0),
		Receipts:     make([]blocks.Block, 0),
		StateNodes:   make(map[common.Hash]blocks.Block),
		StorageNodes: make(map[common.Hash]map[common.Hash]blocks.Block),
	}

	headersErr := f.fetchHeaders(cids, blocks)
	if headersErr != nil {
		return nil, headersErr
	}
	unclesErr := f.fetchUncles(cids, blocks)
	if unclesErr != nil {
		return nil, unclesErr
	}
	trxsErr := f.fetchTrxs(cids, blocks)
	if trxsErr != nil {
		return nil, trxsErr
	}
	rctsErr := f.fetchRcts(cids, blocks)
	if rctsErr != nil {
		return nil, rctsErr
	}
	storageErr := f.fetchStorage(cids, blocks)
	if storageErr != nil {
		return nil, storageErr
	}
	stateErr := f.fetchState(cids, blocks)
	if stateErr != nil {
		return nil, stateErr
	}

	return blocks, nil
}

// fetchHeaders fetches headers
// It uses the f.fetchBatch method
func (f *EthIPLDFetcher) fetchHeaders(cids CIDWrapper, blocks *IPLDWrapper) error {
	log.Debug("fetching header iplds")
	headerCids := make([]cid.Cid, 0, len(cids.Headers))
	for _, c := range cids.Headers {
		dc, err := cid.Decode(c)
		if err != nil {
			return err
		}
		headerCids = append(headerCids, dc)
	}
	blocks.Headers = f.fetchBatch(headerCids)
	if len(blocks.Headers) != len(headerCids) {
		log.Errorf("ipfs fetcher: number of header blocks returned (%d) does not match number expected (%d)", len(blocks.Headers), len(headerCids))
	}
	return nil
}

// fetchUncles fetches uncles
// It uses the f.fetchBatch method
func (f *EthIPLDFetcher) fetchUncles(cids CIDWrapper, blocks *IPLDWrapper) error {
	log.Debug("fetching uncle iplds")
	uncleCids := make([]cid.Cid, 0, len(cids.Uncles))
	for _, c := range cids.Uncles {
		dc, err := cid.Decode(c)
		if err != nil {
			return err
		}
		uncleCids = append(uncleCids, dc)
	}
	blocks.Uncles = f.fetchBatch(uncleCids)
	if len(blocks.Uncles) != len(uncleCids) {
		log.Errorf("ipfs fetcher: number of uncle blocks returned (%d) does not match number expected (%d)", len(blocks.Uncles), len(uncleCids))
	}
	return nil
}

// fetchTrxs fetches transactions
// It uses the f.fetchBatch method
func (f *EthIPLDFetcher) fetchTrxs(cids CIDWrapper, blocks *IPLDWrapper) error {
	log.Debug("fetching transaction iplds")
	trxCids := make([]cid.Cid, 0, len(cids.Transactions))
	for _, c := range cids.Transactions {
		dc, err := cid.Decode(c)
		if err != nil {
			return err
		}
		trxCids = append(trxCids, dc)
	}
	blocks.Transactions = f.fetchBatch(trxCids)
	if len(blocks.Transactions) != len(trxCids) {
		log.Errorf("ipfs fetcher: number of transaction blocks returned (%d) does not match number expected (%d)", len(blocks.Transactions), len(trxCids))
	}
	return nil
}

// fetchRcts fetches receipts
// It uses the f.fetchBatch method
func (f *EthIPLDFetcher) fetchRcts(cids CIDWrapper, blocks *IPLDWrapper) error {
	log.Debug("fetching receipt iplds")
	rctCids := make([]cid.Cid, 0, len(cids.Receipts))
	for _, c := range cids.Receipts {
		dc, err := cid.Decode(c)
		if err != nil {
			return err
		}
		rctCids = append(rctCids, dc)
	}
	blocks.Receipts = f.fetchBatch(rctCids)
	if len(blocks.Receipts) != len(rctCids) {
		log.Errorf("ipfs fetcher: number of receipt blocks returned (%d) does not match number expected (%d)", len(blocks.Receipts), len(rctCids))
	}
	return nil
}

// fetchState fetches state nodes
// It uses the single f.fetch method instead of the batch fetch, because it
// needs to maintain the data's relation to state keys
func (f *EthIPLDFetcher) fetchState(cids CIDWrapper, blocks *IPLDWrapper) error {
	log.Debug("fetching state iplds")
	for _, stateNode := range cids.StateNodes {
		if stateNode.CID == "" || stateNode.Key == "" {
			continue
		}
		dc, decodeErr := cid.Decode(stateNode.CID)
		if decodeErr != nil {
			return decodeErr
		}
		block, fetchErr := f.fetch(dc)
		if fetchErr != nil {
			return fetchErr
		}
		blocks.StateNodes[common.HexToHash(stateNode.Key)] = block
	}
	return nil
}

// fetchStorage fetches storage nodes
// It uses the single f.fetch method instead of the batch fetch, because it
// needs to maintain the data's relation to state and storage keys
func (f *EthIPLDFetcher) fetchStorage(cids CIDWrapper, blks *IPLDWrapper) error {
	log.Debug("fetching storage iplds")
	for _, storageNode := range cids.StorageNodes {
		if storageNode.CID == "" || storageNode.Key == "" || storageNode.StateKey == "" {
			continue
		}
		dc, decodeErr := cid.Decode(storageNode.CID)
		if decodeErr != nil {
			return decodeErr
		}
		blk, fetchErr := f.fetch(dc)
		if fetchErr != nil {
			return fetchErr
		}
		if blks.StorageNodes[common.HexToHash(storageNode.StateKey)] == nil {
			blks.StorageNodes[common.HexToHash(storageNode.StateKey)] = make(map[common.Hash]blocks.Block)
		}
		blks.StorageNodes[common.HexToHash(storageNode.StateKey)][common.HexToHash(storageNode.Key)] = blk
	}
	return nil
}

// fetch is used to fetch a single cid
func (f *EthIPLDFetcher) fetch(cid cid.Cid) (blocks.Block, error) {
	return f.BlockService.GetBlock(context.Background(), cid)
}

// fetchBatch is used to fetch a batch of IPFS data blocks by cid
// There is no guarantee all are fetched, and no error in such a case, so
// downstream we will need to confirm which CIDs were fetched in the result set
func (f *EthIPLDFetcher) fetchBatch(cids []cid.Cid) []blocks.Block {
	fetchedBlocks := make([]blocks.Block, 0, len(cids))
	blockChan := f.BlockService.GetBlocks(context.Background(), cids)
	for block := range blockChan {
		fetchedBlocks = append(fetchedBlocks, block)
	}
	return fetchedBlocks
}
