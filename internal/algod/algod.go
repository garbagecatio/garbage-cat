// Copyright (C) 2022 AlgoNode Org.
//
// algostreamer is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// algostreamer is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with algostreamer.  If not, see <https://www.gnu.org/licenses/>.

package algod

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync/atomic"
	"time"

	"github.com/garbagecatio/rug-ninja-sniper/internal/utils"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/v2/encoding/msgpack"

	"github.com/algorand/go-algorand-sdk/v2/types"
)

type AlgoNodeConfig struct {
	Address string `json:"address"`
	Token   string `json:"token"`
	Id      string `json:"id"`
}

type AlgoConfig struct {
	ANodes []*AlgoNodeConfig `json:"nodes"`
	Queue  int               `json:"queue"`
	FRound int64             `json:"first"`
	LRound int64             `json:"last"`
}

type Status struct {
	LastRound uint64
	LagMs     int64
	NodeId    string
	LastCP    string
}

type BlockWrap struct {
	Block    *types.Block `json: "block"`
	BlockRaw []byte       `json:"-"`
	Src      string       `json:"src"`
	Ts       time.Time    `json:"ts"`
}

// globalMaxBlock holds the highest read block across all connected nodes
// writes must use atomic interface
// reads are safe as the var is 64bit aligned
var globalMaxBlock uint64 = 0

func AlgoStreamer(ctx context.Context, acfg *AlgoConfig) (chan *BlockWrap, chan *Status, error) {
	qDepth := acfg.Queue
	if qDepth < 1 {
		qDepth = 100
	}
	bestbchan := make(chan *BlockWrap, qDepth)
	bchan := make(chan *BlockWrap, qDepth)
	schan := make(chan *Status, qDepth)

	for idx := range acfg.ANodes {
		if err := algodStreamNode(ctx, acfg, idx, bchan, schan, acfg.FRound, acfg.LRound); err != nil {
			return nil, nil, err
		}
	}

	// filter duplicates, forward only first newer blocks.
	go func() {
		var maxBlock uint64 = math.MaxUint64
		var maxTs time.Time = time.Now()
		var maxLeader string = "'"
		for {
			select {
			case bw := <-bchan:
				if uint64(bw.Block.Round) > maxBlock || maxBlock == math.MaxUint64 {
					bestbchan <- bw
					maxBlock = uint64(bw.Block.Round)
					atomic.StoreUint64(&globalMaxBlock, maxBlock)
					maxTs = bw.Ts
					maxLeader = bw.Src
				} else {
					if maxBlock == uint64(bw.Block.Round) {
						fmt.Fprintf(os.Stderr, "[INFO][ALGOD] Block from %s is %v behind %s\n", bw.Src, bw.Ts.Sub(maxTs), maxLeader)
					}
				}
			case <-ctx.Done():
			}
		}
	}()

	return bestbchan, schan, nil
}

func algodStreamNode(ctx context.Context, acfg *AlgoConfig, idx int, bchan chan *BlockWrap, schan chan *Status, start int64, stop int64) error {

	cfg := acfg.ANodes[idx]
	// Create an algod client
	algodClient, err := algod.MakeClient(cfg.Address, cfg.Token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!ERR][ALGOD][%s] failed to make algod client: %s\n", cfg.Id, err)
		return err
	}
	fmt.Fprintf(os.Stderr, "[INFO][ALGOD][%s] new algod client: %s\n", cfg.Id, cfg.Address)

	//Loop until Algoverse gets cancelled
	go func() {

		var nodeStatus *models.NodeStatus = nil
		utils.Backoff(ctx, func(actx context.Context) error {
			fmt.Printf("[INFO][ALGOD][%s] Getting node status", cfg.Id)
			ns, err := algodClient.Status().Do(actx)
			if err != nil {
				return fmt.Errorf("[!ERR][ALGOD][%s] %s\n", cfg.Id, err.Error())
			}
			nodeStatus = &ns
			return nil
		}, time.Second*10, time.Millisecond*100, time.Second*10)
		if nodeStatus == nil {
			fmt.Fprintf(os.Stderr, "[!ERR][ALGOD][%s] Unable to start node\n", cfg.Id)
			return
		}
		schan <- &Status{NodeId: cfg.Id, LastCP: nodeStatus.LastCatchpoint, LastRound: uint64(nodeStatus.LastRound), LagMs: int64(nodeStatus.TimeSinceLastRound) / int64(time.Millisecond)}

		var nextRound uint64 = 0
		if start < 0 {
			nextRound = nodeStatus.LastRound
			fmt.Fprintf(os.Stderr, "[WARN][ALGOD][%s] Starting from last round : %d\n", cfg.Id, nodeStatus.LastRound)
		} else {
			nextRound = uint64(start)
			fmt.Fprintf(os.Stderr, "[WARN][ALGOD][%s] Starting from fixed round : %d\n", cfg.Id, nextRound)
		}

		ustop := uint64(stop)
		for stop < 0 || nextRound <= ustop {
			for ; nextRound <= nodeStatus.LastRound; nextRound++ {
				err := utils.Backoff(ctx, func(actx context.Context) error {
					gMax := globalMaxBlock
					//skip old blocks in case other nodes are ahead of us
					if gMax > nextRound {
						fmt.Fprintf(os.Stderr, "[WARN][ALGOD][%s] skipping ahead %d blocks to %d\n", cfg.Id, gMax-nextRound, gMax)
						nextRound = globalMaxBlock
					}
					rawBlock, err := algodClient.BlockRaw(nextRound).Do(ctx)
					if err != nil {
						return fmt.Errorf("[!ERR][ALGOD][%s] %s", cfg.Id, err.Error())
					}
					var response models.BlockResponse
					msgpack.CodecHandle.ErrorIfNoField = false
					if err = msgpack.Decode(rawBlock, &response); err != nil {
						return fmt.Errorf("[!ERR][ALGOD][%s] %s", cfg.Id, err.Error())
					}
					block := response.Block

					//fmt.Fprintf(os.Stderr, "got block %d, queue %d\n", block.Round, len(bchan))
					select {
					case bchan <- &BlockWrap{
						Block:    &block,
						BlockRaw: rawBlock,
						Ts:       time.Now(),
						Src:      cfg.Id,
					}:
					case <-ctx.Done():
					}
					return ctx.Err()
				}, time.Second*10, time.Millisecond*100, time.Second*10)
				if err != nil || nextRound >= ustop {
					return
				}
			}

			err := utils.Backoff(ctx, func(actx context.Context) error {
				newStatus, err := algodClient.StatusAfterBlock(nodeStatus.LastRound).Do(actx)
				if err != nil {
					return fmt.Errorf("[!ERR][ALGOD][%s] %s", cfg.Id, err.Error())
				}
				nodeStatus = &newStatus
				//fmt.Fprintf(os.Stderr, "algod last round: %d, lag: %s\n", nodeStatus.LastRound, time.Duration(nodeStatus.TimeSinceLastRound)*time.Nanosecond)
				schan <- &Status{NodeId: cfg.Id, LastRound: uint64(nodeStatus.LastRound), LagMs: int64(nodeStatus.TimeSinceLastRound) / int64(time.Millisecond)}
				return nil
			}, time.Second*10, time.Millisecond*100, time.Second*10)

			if err != nil {
				return
			}

		}
	}()

	return nil
}
