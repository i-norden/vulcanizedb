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
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/statediff"
	log "github.com/sirupsen/logrus"

	"github.com/vulcanize/vulcanizedb/pkg/config"
	"github.com/vulcanize/vulcanizedb/pkg/core"
	"github.com/vulcanize/vulcanizedb/pkg/datastore/postgres"
)

const payloadChanBufferSize = 20000 // the max eth sub buffer size

// SyncPublishScreenAndServe is the top level interface for streaming, converting to IPLDs, publishing,
// and indexing all Ethereum data; screening this data; and serving it up to subscribed clients
// This service is compatible with the Ethereum service interface (node.Service)
type SyncPublishScreenAndServe interface {
	// APIs(), Protocols(), Start() and Stop()
	node.Service
	// Main event loop for syncAndPublish processes
	SyncAndPublish(wg *sync.WaitGroup, forwardPayloadChan chan<- IPLDPayload, forwardQuitchan chan<- bool) error
	// Main event loop for handling client pub-sub
	ScreenAndServe(wg *sync.WaitGroup, receivePayloadChan <-chan IPLDPayload, receiveQuitchan <-chan bool)
	// Method to subscribe to receive state diff processing output
	Subscribe(id rpc.ID, sub chan<- ResponsePayload, quitChan chan<- bool, streamFilters config.Subscription)
	// Method to unsubscribe from state diff processing
	Unsubscribe(id rpc.ID)
}

// Service is the underlying struct for the SyncAndPublish interface
type Service struct {
	// Used to sync access to the Subscriptions
	sync.Mutex
	// Interface for streaming statediff payloads over a geth rpc subscription
	Streamer StateDiffStreamer
	// Interface for converting statediff payloads into ETH-IPLD object payloads
	Converter PayloadConverter
	// Interface for publishing the ETH-IPLD payloads to IPFS
	Publisher IPLDPublisher
	// Interface for indexing the CIDs of the published ETH-IPLDs in Postgres
	Repository CIDRepository
	// Interface for filtering and serving data according to subscribed clients according to their specification
	Screener ResponseScreener
	// Interface for fetching ETH-IPLD objects from IPFS
	Fetcher IPLDFetcher
	// Interface for searching and retrieving CIDs from Postgres index
	Retriever CIDRetriever
	// Interface for resolving ipfs blocks to their data types
	Resolver IPLDResolver
	// Chan the processor uses to subscribe to state diff payloads from the Streamer
	PayloadChan chan statediff.Payload
	// Used to signal shutdown of the service
	QuitChan chan bool
	// A mapping of rpc.IDs to their subscription channels, mapped to their subscription type (hash of the StreamFilters)
	Subscriptions map[common.Hash]map[rpc.ID]Subscription
	// A mapping of subscription hash type to the corresponding StreamFilters
	SubscriptionTypes map[common.Hash]config.Subscription
}

// NewIPFSProcessor creates a new Processor interface using an underlying Processor struct
func NewIPFSProcessor(ipfsPath string, db *postgres.DB, ethClient core.EthClient, rpcClient core.RpcClient, qc chan bool) (SyncPublishScreenAndServe, error) {
	publisher, err := NewIPLDPublisher(ipfsPath)
	if err != nil {
		return nil, err
	}
	fetcher, err := NewIPLDFetcher(ipfsPath)
	if err != nil {
		return nil, err
	}
	return &Service{
		Streamer:          NewStateDiffStreamer(rpcClient),
		Repository:        NewCIDRepository(db),
		Converter:         NewPayloadConverter(ethClient),
		Publisher:         publisher,
		Screener:          NewResponseScreener(),
		Fetcher:           fetcher,
		Retriever:         NewCIDRetriever(db),
		Resolver:          NewIPLDResolver(),
		PayloadChan:       make(chan statediff.Payload, payloadChanBufferSize),
		QuitChan:          qc,
		Subscriptions:     make(map[common.Hash]map[rpc.ID]Subscription),
		SubscriptionTypes: make(map[common.Hash]config.Subscription),
	}, nil
}

// Protocols exports the services p2p protocols, this service has none
func (sap *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns the RPC descriptors the StateDiffingService offers
func (sap *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: APIName,
			Version:   APIVersion,
			Service:   NewPublicSeedNodeAPI(sap),
			Public:    true,
		},
	}
}

// SyncAndPublish is the backend processing loop which streams data from geth, converts it to iplds, publishes them to ipfs, and indexes their cids
// This continues on no matter if or how many subscribers there are, it then forwards the data to the ScreenAndServe() loop
// which filters and sends relevant data to client subscriptions, if there are any
func (sap *Service) SyncAndPublish(wg *sync.WaitGroup, forwardPayloadChan chan<- IPLDPayload, forwardQuitchan chan<- bool) error {
	sub, err := sap.Streamer.Stream(sap.PayloadChan)
	if err != nil {
		return err
	}
	wg.Add(1)
	go func() {
		for {
			select {
			case payload := <-sap.PayloadChan:
				if payload.Err != nil {
					log.Error(err)
					continue
				}
				ipldPayload, err := sap.Converter.Convert(payload)
				if err != nil {
					log.Error(err)
					continue
				}
				// If we have a ScreenAndServe process running, forward the payload to it
				select {
				case forwardPayloadChan <- *ipldPayload:
				default:
				}
				cidPayload, err := sap.Publisher.Publish(ipldPayload)
				if err != nil {
					log.Error(err)
					continue
				}
				err = sap.Repository.Index(cidPayload)
				if err != nil {
					log.Error(err)
				}
			case err = <-sub.Err():
				log.Error(err)
			case <-sap.QuitChan:
				// If we have a ScreenAndServe process running, forward the quit signal to it
				select {
				case forwardQuitchan <- true:
				default:
				}
				log.Info("quiting SyncAndPublish process")
				wg.Done()
				return
			}
		}
	}()

	return nil
}

// ScreenAndServe is the loop used to screen data streamed from the state diffing eth node
// and send the appropriate portions of it to a requesting client subscription, according to their subscription configuration
func (sap *Service) ScreenAndServe(wg *sync.WaitGroup, receivePayloadChan <-chan IPLDPayload, receiveQuitchan <-chan bool) {
	wg.Add(1)
	go func() {
		for {
			select {
			case payload := <-receivePayloadChan:
				err := sap.processResponse(payload)
				if err != nil {
					log.Error(err)
				}
			case <-receiveQuitchan:
				log.Info("quiting ScreenAndServe process")
				wg.Done()
				return
			}
		}
	}()
}

func (sap *Service) processResponse(payload IPLDPayload) error {
	for ty, subs := range sap.Subscriptions {
		// Retreive the subscription paramaters for this subscription type
		subConfig, ok := sap.SubscriptionTypes[ty]
		if !ok {
			return fmt.Errorf("subscription configuration for subscription type %s not available", ty.Hex())
		}
		response, err := sap.Screener.ScreenResponse(subConfig, payload)
		if err != nil {
			return err
		}
		for id := range subs {
			//TODO send payloads to this type of sub
			sap.serve(id, *response, ty)

		}
	}
	return nil
}

// Subscribe is used by the API to subscribe to the service loop
func (sap *Service) Subscribe(id rpc.ID, sub chan<- ResponsePayload, quitChan chan<- bool, streamFilters config.Subscription) {
	log.Info("Subscribing to the seed node service")
	// Subscription type is defined as the hash of its content
	// Group subscriptions by type and screen payloads once for subs of the same type
	by, err := rlp.EncodeToBytes(streamFilters)
	if err != nil {
		log.Error(err)
	}
	subscriptionHash := crypto.Keccak256(by)
	subscriptionType := common.BytesToHash(subscriptionHash)
	subscription := Subscription{
		PayloadChan: sub,
		QuitChan:    quitChan,
	}
	// If the subscription requests a backfill, use the Postgres index to lookup and retrieve historical data
	// Otherwise we only filter new data as it is streamed in from the state diffing geth node
	if streamFilters.BackFill || streamFilters.BackFillOnly {
		sap.backFill(subscription, id, streamFilters)
	}
	if !streamFilters.BackFillOnly {
		sap.Lock()
		if sap.Subscriptions[subscriptionType] == nil {
			sap.Subscriptions[subscriptionType] = make(map[rpc.ID]Subscription)
		}
		sap.Subscriptions[subscriptionType][id] = subscription
		sap.SubscriptionTypes[subscriptionType] = streamFilters
		sap.Unlock()
	}
}

func (sap *Service) backFill(sub Subscription, id rpc.ID, con config.Subscription) {
	log.Debug("back-filling data for id", id)
	// Retrieve cached CIDs relevant to this subscriber
	cidWrappers, err := sap.Retriever.RetrieveCIDs(con)
	if err != nil {
		sub.PayloadChan <- ResponsePayload{
			ErrMsg: "CID retrieval error: " + err.Error(),
		}
	}
	for _, cidWrapper := range cidWrappers {
		blocksWrapper, err := sap.Fetcher.FetchCIDs(cidWrapper)
		if err != nil {
			log.Error(err)
			sub.PayloadChan <- ResponsePayload{
				ErrMsg: "IPLD fetching error: " + err.Error(),
			}
		}
		backFillIplds, err := sap.Resolver.ResolveIPLDs(*blocksWrapper)
		if err != nil {
			log.Error(err)
			sub.PayloadChan <- ResponsePayload{
				ErrMsg: "IPLD resolving error: " + err.Error(),
			}
		}
		select {
		case sub.PayloadChan <- *backFillIplds:
			log.Infof("sending seed node back-fill payload to subscription %s", id)
		default:
			log.Infof("unable to send back-fill ppayload to subscription %s; channel has no receiver", id)
		}
	}
}

// Unsubscribe is used to unsubscribe to the StateDiffingService loop
func (sap *Service) Unsubscribe(id rpc.ID) {
	log.Info("Unsubscribing from the seed node service")
	sap.Lock()
	for ty := range sap.Subscriptions {
		delete(sap.Subscriptions[ty], id)
		if len(sap.Subscriptions[ty]) == 0 {
			// If we removed the last subscription of this type, remove the subscription type outright
			delete(sap.Subscriptions, ty)
			delete(sap.SubscriptionTypes, ty)
		}
	}
	sap.Unlock()
}

// Start is used to begin the service
func (sap *Service) Start(*p2p.Server) error {
	log.Info("Starting seed node service")
	wg := new(sync.WaitGroup)
	payloadChan := make(chan IPLDPayload, payloadChanBufferSize)
	quitChan := make(chan bool, 1)
	if err := sap.SyncAndPublish(wg, payloadChan, quitChan); err != nil {
		return err
	}
	sap.ScreenAndServe(wg, payloadChan, quitChan)
	return nil
}

// Stop is used to close down the service
func (sap *Service) Stop() error {
	log.Info("Stopping seed node service")
	close(sap.QuitChan)
	return nil
}

// serve is used to send screened payloads to their requesting sub
func (sap *Service) serve(id rpc.ID, payload ResponsePayload, ty common.Hash) {
	sap.Lock()
	sub, ok := sap.Subscriptions[ty][id]
	if ok {
		select {
		case sub.PayloadChan <- payload:
			log.Infof("sending seed node payload to subscription %s", id)
		default:
			log.Infof("unable to send payload to subscription %s; channel has no receiver", id)
		}
	}
	sap.Unlock()
}

// close is used to close all listening subscriptions
func (sap *Service) close() {
	sap.Lock()
	for ty, subs := range sap.Subscriptions {
		for id, sub := range subs {
			select {
			case sub.QuitChan <- true:
				log.Infof("closing subscription %s", id)
			default:
				log.Infof("unable to close subscription %s; channel has no receiver", id)
			}
		}
		delete(sap.Subscriptions, ty)
		delete(sap.SubscriptionTypes, ty)
	}
	sap.Unlock()
}