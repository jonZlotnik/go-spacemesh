package types

import (
	"crypto/sha512"
)

// txSelf is an interface to encode/decode transaction header or access to self Transaction interface
type txSelf interface {
	xdrBytes() ([]byte, error)
	xdrFill([]byte) (int, error)
	complete() *CommonTx
}

type txMutable interface {
	immutableBytes() ([]byte, error)
}

// IncompleteCommonTx is a common partial implementation of the IncompleteTransaction
type IncompleteCommonTx struct {
	self   txSelf
	txType TransactionType
}

// AuthenticationMessage returns the authentication message for the transaction
func (t IncompleteCommonTx) AuthenticationMessage() (txm TransactionAuthenticationMessage, err error) {
	txm.TransactionData, err = t.self.xdrBytes()
	if err != nil {
		return
	}
	sha := sha512.New()
	networkID := GetNetworkID()
	_, _ = sha.Write(networkID[:])
	_, _ = sha.Write([]byte{byte(t.txType)})
	if p, ok := t.self.(txMutable); ok {
		d, err := p.immutableBytes() // err shadowed
		if err != nil {
			return txm, err
		}
		_, _ = sha.Write(d)
	} else {
		_, _ = sha.Write(txm.TransactionData)
	}
	copy(txm.Digest[:], sha.Sum(nil))
	txm.TxType = t.txType
	return
}

// Type returns transaction's type
func (t IncompleteCommonTx) Type() TransactionType {
	return t.txType
}

// Complete converts the IncompleteTransaction to the Transaction object
func (t *IncompleteCommonTx) Complete(pubKey TxPublicKey, signature TxSignature, txid TransactionID) Transaction {
	tx := t.self.complete()
	tx.txType = t.txType
	tx.origin = BytesToAddress(pubKey.Bytes())
	tx.id = txid
	tx.pubKey = pubKey
	tx.signature = signature
	return tx.self.(Transaction)
}

// decode fills the transaction object from the transaction bytes
func (t *IncompleteCommonTx) decode(data []byte, txtp TransactionType) (err error) {
	var n int
	if n, err = t.self.xdrFill(data); err != nil {
		return
	}
	if n != len(data) {
		// to protect against digest compilation attack with ED++
		//   app call/spawn txs can be vulnerable because variable call-data size
		return errBadTransactionEncodingError
	}
	t.txType = txtp
	return
}

// CommonTx is an common partial implementation of the Transaction
type CommonTx struct {
	IncompleteCommonTx
	origin    Address
	id        TransactionID
	signature TxSignature
	pubKey    TxPublicKey
}

// Origin returns transaction's origin, it implements Transaction.Origin
func (tx CommonTx) Origin() Address {
	return tx.origin
}

// Signature returns transaction's signature, it implements Transaction.Signature
func (tx CommonTx) Signature() TxSignature {
	return tx.signature
}

// PubKey returns transaction's public key, it implements Transaction.PubKey
func (tx CommonTx) PubKey() TxPublicKey {
	return tx.pubKey
}

// ID returns the transaction's ID. , it implements Transaction.ID
func (tx *CommonTx) ID() TransactionID {
	return tx.id
}

// Hash32 returns the TransactionID as a Hash32.
func (tx *CommonTx) Hash32() Hash32 {
	return tx.id.Hash32()
}

// ShortString returns a the first 5 characters of the ID, for logging purposes.
func (tx *CommonTx) ShortString() string {
	return tx.id.ShortString()
}

// Encode encodes the transaction to a signed transaction. It implements Transaction.Encode
func (tx *CommonTx) Encode() (_ SignedTransaction, err error) {
	txm, err := tx.AuthenticationMessage()
	if err != nil {
		return
	}
	return txm.Encode(tx.pubKey, tx.signature)
}

// Prune by default does nothing and returns original transaction
func (tx *CommonTx) Prune() Transaction {
	return tx.self.(Transaction)
}

// Pruned returns true if transaction is pruned, by default transaction is not prunable
func (tx *CommonTx) Pruned() bool {
	return false
}
