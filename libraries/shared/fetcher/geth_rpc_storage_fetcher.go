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
package fetcher

import (
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/statediff"
	"github.com/sirupsen/logrus"

	"github.com/vulcanize/vulcanizedb/libraries/shared/storage/utils"
	"github.com/vulcanize/vulcanizedb/libraries/shared/streamer"
)

type GethRpcStorageFetcher struct {
	statediffPayloadChan chan statediff.Payload
	streamer             streamer.Streamer
}

func NewGethRpcStorageFetcher(streamer streamer.Streamer, statediffPayloadChan chan statediff.Payload) GethRpcStorageFetcher {
	return GethRpcStorageFetcher{
		statediffPayloadChan: statediffPayloadChan,
		streamer:             streamer,
	}
}

func (fetcher GethRpcStorageFetcher) FetchStorageDiffs(out chan<- utils.StorageDiff, errs chan<- error) {
	ethStatediffPayloadChan := fetcher.statediffPayloadChan
	clientSubscription, clientSubErr := fetcher.streamer.Stream(ethStatediffPayloadChan)
	if clientSubErr != nil {
		errs <- clientSubErr
		panic(fmt.Sprintf("Error creating a geth client subscription: %v", clientSubErr))
	}
	logrus.Info("Successfully created a geth client subscription: ", clientSubscription)

	for {
		diff := <-ethStatediffPayloadChan
		logrus.Trace("received a statediff")
		stateDiff := new(statediff.StateDiff)
		decodeErr := rlp.DecodeBytes(diff.StateDiffRlp, stateDiff)
		if decodeErr != nil {
			logrus.Warn("Error decoding state diff into RLP: ", decodeErr)
			errs <- decodeErr
		}

		accounts := utils.GetAccountsFromDiff(*stateDiff)
		logrus.Trace(fmt.Sprintf("iterating through %d accounts on stateDiff for block %d", len(accounts), stateDiff.BlockNumber))
		for _, account := range accounts {
			logrus.Trace(fmt.Sprintf("iterating through %d Storage values on account", len(account.Storage)))
			for _, storage := range account.Storage {
				diff, formatErr := utils.FromGethStateDiff(account, stateDiff, storage)
				if formatErr != nil {
					errs <- formatErr
					continue
				}
				logrus.Trace("adding storage diff to out channel",
					"keccak of address: ", diff.HashedAddress.Hex(),
					"block height: ", diff.BlockHeight,
					"storage key: ", diff.StorageKey.Hex(),
					"storage value: ", diff.StorageValue.Hex())

				out <- diff
			}
		}
	}
}
