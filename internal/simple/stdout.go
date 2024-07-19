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

package simple

import (
	"context"
	"fmt"

	"github.com/garbagecatio/rug-ninja-sniper/internal/algod"

	"github.com/algorand/go-algorand/protocol"

	"github.com/algorand/go-codec/codec"
)

func handleBlockStdOut(b *algod.BlockWrap) error {
	var output []byte
	enc := codec.NewEncoderBytes(&output, protocol.JSONStrictHandle)
	err := enc.Encode(b)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func SimplePusher(ctx context.Context, blocks chan *algod.BlockWrap, status chan *algod.Status) error {

	go func() {
		for {
			select {
			case <-status:
				//noop
			case b := <-blocks:
				handleBlockStdOut(b)
			case <-ctx.Done():
			}

		}
	}()
	return nil
}
