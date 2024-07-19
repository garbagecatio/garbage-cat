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

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/algorand/go-algorand/protocol"
	"github.com/algorand/go-codec/codec"
	"github.com/tidwall/jsonc"
)

type eternalFn func(ctx context.Context) error

func Backoff(ctx context.Context, fn eternalFn, timeout time.Duration, wait time.Duration, maxwait time.Duration) error {
	//Loop until Algoverse gets cancelled
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		cctx, cancel := context.WithTimeout(ctx, timeout)
		err := fn(cctx)
		if err == nil { // Success
			cancel()
			return nil
		}
		cancel()
		fmt.Fprintf(os.Stderr, err.Error())

		//keep an eye on cancellation while backing off
		if wait > 0 {
			select {
			case <-ctx.Done():
			case <-time.After(wait):
			}
			if maxwait > 0 {
				wait *= 2
				if wait > maxwait {
					wait = maxwait
				}
			}
		}
	}
}

func LoadJSONCFromFile(filename string, object interface{}) (err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonc.ToJSON(data), &object)
}

func EncodeJson(obj interface{}) ([]byte, error) {
	var output []byte
	enc := codec.NewEncoderBytes(&output, protocol.JSONStrictHandle)

	err := enc.Encode(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to encode object: %v", err)
	}
	return output, nil
}
