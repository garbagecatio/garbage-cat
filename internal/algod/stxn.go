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
	"fmt"

	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/types"
	"github.com/algorand/go-algorand/config"
	"github.com/algorand/go-algorand/protocol"
)

func DecodeTxnId(bh types.BlockHeader, stb *types.SignedTxnInBlock) (string, error) {
	st := &stb.SignedTxn

	proto, ok := config.Consensus[protocol.ConsensusVersion(bh.CurrentProtocol)]
	if !ok {
		return "", fmt.Errorf("consensus protocol %s not found", bh.CurrentProtocol)
	}
	if !proto.SupportSignedTxnInBlock {
		return "", nil
	}

	if st.Txn.GenesisID != "" {
		return "", fmt.Errorf("GenesisID <%s> not empty", st.Txn.GenesisID)
	}

	if stb.HasGenesisID {
		st.Txn.GenesisID = bh.GenesisID
	}

	if st.Txn.GenesisHash != (types.Digest{}) {
		return "", fmt.Errorf("GenesisHash <%v> not empty", st.Txn.GenesisHash)
	}

	if proto.RequireGenesisHash {
		if stb.HasGenesisHash {
			return "", fmt.Errorf("HasGenesisHash set to true but RequireGenesisHash obviates the flag")
		}
		st.Txn.GenesisHash = bh.GenesisHash
	} else {
		if stb.HasGenesisHash {
			st.Txn.GenesisHash = bh.GenesisHash
		}
	}

	return crypto.GetTxID(st.Txn), nil
}
