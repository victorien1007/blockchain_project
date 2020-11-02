package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"

	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
)

const subsidy = 10

// Transaction represents a Bitcoin transaction
type Transaction struct {
	Id   []byte
	Vin  []TransInput
	Vout []TransOutput
}


// TransInput represents a transaction input
type TransInput struct {
	Id      []byte
	Vout      int //Value out
	Sign []byte //Signature
	PubK    []byte //PublicKey
}

// TransOutput represents a transaction output
type TransOutput struct {
	V      int //Value
	PubKH 	[]byte//PublicKeyHash
}

// UsesKey checks whether the address initiated the transaction
func (in *TransInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PubK)
	return bytes.Compare(lockingHash, pubKeyHash) == 0
}


// Lock signs the output
func (out *TransOutput) Lock(address []byte) {
	pKH := Base58Decode(address)
	pKH = pKH[1 : len(pKH)-4]
	out.PubKH = pKH
}

// IsLockedWithKey checks if the output can be used by the owner of the pubkey
func (out *TransOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKH, pubKeyHash) == 0
}

// NewTransOutput create a new TransOutput
func NewTransOutput(value int, address string) *TransOutput {
	transout := &TransOutput{value, nil}
	transout.Lock([]byte(address))

	return transout
}

// IsCoinbase checks whether the transaction is coinbase
func (trans Transaction) IsCoinbase() bool {
	return len(trans.Vin) == 1 && len(trans.Vin[0].Id) == 0 && trans.Vin[0].Vout == -1
}

// Serialize returns a serialized Transaction
func (trans Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(trans)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

// Hash returns the hash of the Transaction
func (trans *Transaction) Hash() []byte {
	var hash [32]byte

	transCopy := *trans
	transCopy.Id = []byte{}

	hash = sha256.Sum256(transCopy.Serialize())

	return hash[:]
}

// Sign signs each input of a Transaction
func (trans *Transaction) Sign(privKey ecdsa.PrivateKey, prevTrans map[string]Transaction) {
	if trans.IsCoinbase() {
		return
	}

	for _, vin := range trans.Vin {
		if prevTrans[hex.EncodeToString(vin.Id)].Id == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	transCopy := trans.TrimmedCopy()

	for inID, vin := range transCopy.Vin {
		prevTran := prevTrans[hex.EncodeToString(vin.Id)]
		transCopy.Vin[inID].Sign = nil
		transCopy.Vin[inID].PubK = prevTran.Vout[vin.Vout].PubKH
		transCopy.Id = transCopy.Hash()
		transCopy.Vin[inID].PubK = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, transCopy.Id)
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		trans.Vin[inID].Sign = signature
	}
}

// String returns a human-readable representation of a transaction
func (trans Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", trans.Id))

	for i, input := range trans.Vin {

		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       ID:      %x", input.Id))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Sign))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubK))
	}

	for i, output := range trans.Vout {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.V))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKH))
	}

	return strings.Join(lines, "\n")
}

// TrimmedCopy creates a trimmed copy of Transaction to be used in signing
func (trans *Transaction) TrimmedCopy() Transaction {
	var inputs []TransInput
	var outputs []TransOutput

	for _, vin := range trans.Vin {
		inputs = append(inputs, TransInput{vin.Id, vin.Vout, nil, nil})
	}

	for _, vout := range trans.Vout {
		outputs = append(outputs, TransOutput{vout.V, vout.PubKH})
	}

	transCopy := Transaction{trans.Id, inputs, outputs}

	return transCopy
}

// Verify verifies signatures of Transaction inputs
func (trans *Transaction) Verify(prevTrans map[string]Transaction) bool {
	if trans.IsCoinbase() {
		return true
	}

	for _, vin := range trans.Vin {
		if prevTrans[hex.EncodeToString(vin.Id)].Id == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	transCopy := trans.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range trans.Vin {
		prevTran := prevTrans[hex.EncodeToString(vin.Id)]
		transCopy.Vin[inID].Sign = nil
		transCopy.Vin[inID].PubK = prevTran.Vout[vin.Vout].PubKH
		transCopy.Id = transCopy.Hash()
		transCopy.Vin[inID].PubK = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Sign)
		r.SetBytes(vin.Sign[:(sigLen / 2)])
		s.SetBytes(vin.Sign[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubK)
		x.SetBytes(vin.PubK[:(keyLen / 2)])
		y.SetBytes(vin.PubK[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, transCopy.Id, &r, &s) == false {
			return false
		}
	}

	return true
}

// NewCoinTrans creates a new coinbase transaction
func NewCoinTrans(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}

	in := TransInput{[]byte{}, -1, nil, []byte(data)}
	out := NewTransOutput(subsidy, to)
	trans := Transaction{nil, []TransInput{in}, []TransOutput{*out}}
	trans.Id = trans.Hash()

	return &trans
}

// NewTransaction creates a new transaction
func NewTransaction(from, to string, amount int, bc *Blockchain) *Transaction {
	var inputs []TransInput
	var outputs []TransOutput

	wallets, err := NewWallets()
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)
	pubKH := HashPubKey(wallet.PubK)
	acc, validOutputs := bc.FindOutputs(pubKH, amount)

	if acc < amount {
		log.Panic("ERROR: Not enough funds")
	}

	// Build a list of inputs
	for transid, outs := range validOutputs {
		transID, err := hex.DecodeString(transid)
		if err != nil {
			log.Panic(err)
		}

		for _, out := range outs {
			input := TransInput{transID, out, nil, wallet.PubK}
			inputs = append(inputs, input)
		}
	}

	// Build a list of outputs
	outputs = append(outputs, *NewTransOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *NewTransOutput(acc-amount, from)) // a change
	}

	trans := Transaction{nil, inputs, outputs}
	trans.Id = trans.Hash()
	bc.SignTransaction(&trans, wallet.PriK)

	return &trans
}
