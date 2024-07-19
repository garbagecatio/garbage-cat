package misc

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/algorand/go-algorand-sdk/v2/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

func ListInner(stxn *types.SignedTxnWithAD) []types.SignedTxnWithAD {
	txns := []types.SignedTxnWithAD{}
	for _, itxn := range stxn.ApplyData.EvalDelta.InnerTxns {
		txns = append(txns, itxn)
		txns = append(txns, ListInner(&itxn)...)
	}
	return txns
}

func AddrToED25519PublicKey(a types.Address) (pk ed25519.PublicKey) {
	pk = make([]byte, len(a))
	copy(pk, a[:])
	return
}

func RawTransactionBytesToSign(tx types.Transaction) []byte {
	// Encode the transaction as msgpack
	encodedTx := msgpack.Encode(tx)

	// Prepend the hashable prefix
	msgParts := [][]byte{[]byte("TX"), encodedTx}
	return bytes.Join(msgParts, nil)
}

func VerifySignature(tx types.Transaction, pk ed25519.PublicKey, sig types.Signature) bool {
	toBeSigned := RawTransactionBytesToSign(tx)
	return ed25519.Verify(pk, toBeSigned, sig[:])
}

// CheckSignature verifies that stx is either a single signature
func CheckSignature(stx types.SignedTxn) error {
	if stx.Sig == (types.Signature{}) {
		return errors.New("msig/lsig not supported")
	}

	// ensure other signature fields are empty
	if len(stx.Msig.Subsigs) != 0 || stx.Msig.Version != 0 || stx.Msig.Threshold != 0 {
		return errors.New("tx has both a sig and msig")
	}

	if !stx.Lsig.Blank() {
		return errors.New("tx has both a sig and lsig")
	}

	fmt.Println("Auth Address: ", stx.AuthAddr.String())

	var pk ed25519.PublicKey
	if stx.AuthAddr.String() != types.ZeroAddress.String() {
		pk = AddrToED25519PublicKey(stx.AuthAddr)
	} else {
		pk = AddrToED25519PublicKey(stx.Txn.Sender)
	}

	if !VerifySignature(stx.Txn, pk, stx.Sig) {
		return errors.New("signature is invalid")
	}

	return nil
}
