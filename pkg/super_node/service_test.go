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

package super_node_test

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/statediff"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mocks2 "github.com/vulcanize/vulcanizedb/libraries/shared/mocks"
	"github.com/vulcanize/vulcanizedb/pkg/ipfs/mocks"
	"github.com/vulcanize/vulcanizedb/pkg/super_node"
	mocks3 "github.com/vulcanize/vulcanizedb/pkg/super_node/mocks"
)

var _ = Describe("Service", func() {
	Describe("SyncAndPublish", func() {
		It("Streams statediff.Payloads, converts them to IPLDPayloads, publishes IPLDPayloads, and indexes CIDPayloads", func() {
			wg := new(sync.WaitGroup)
			payloadChan := make(chan statediff.Payload, 1)
			quitChan := make(chan bool, 1)
			mockCidRepo := &mocks3.CIDRepository{
				ReturnErr: nil,
			}
			mockPublisher := &mocks.IPLDPublisher{
				ReturnCIDPayload: mocks.MockCIDPayload,
				ReturnErr:        nil,
			}
			mockStreamer := &mocks2.StateDiffStreamer{
				ReturnSub: &rpc.ClientSubscription{},
				StreamPayloads: []statediff.Payload{
					mocks.MockStateDiffPayload,
				},
				ReturnErr: nil,
			}
			mockConverter := &mocks.PayloadConverter{
				ReturnIPLDPayload: mocks.MockIPLDPayload,
				ReturnErr:         nil,
			}
			processor := &super_node.Service{
				Repository:     mockCidRepo,
				Publisher:      mockPublisher,
				Streamer:       mockStreamer,
				Converter:      mockConverter,
				PayloadChan:    payloadChan,
				QuitChan:       quitChan,
				WorkerPoolSize: 1,
			}
			err := processor.SyncAndPublish(wg, nil, nil)
			Expect(err).ToNot(HaveOccurred())
			time.Sleep(2 * time.Second)
			quitChan <- true
			wg.Wait()
			Expect(mockConverter.PassedStatediffPayload).To(Equal(mocks.MockStateDiffPayload))
			Expect(len(mockCidRepo.PassedCIDPayload)).To(Equal(1))
			Expect(mockCidRepo.PassedCIDPayload[0]).To(Equal(mocks.MockCIDPayload))
			Expect(mockPublisher.PassedIPLDPayload).To(Equal(mocks.MockIPLDPayload))
			Expect(mockStreamer.PassedPayloadChan).To(Equal(payloadChan))
		})
	})
})
