package spec

import (
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// from core/types/bock.go
type extblock struct {
	Header *types.Header
	Txs    []*types.Transaction
	Uncles []*types.Header
}

func fillHdr(eh *ExecutionHeader) *types.Header {
	hdr := types.Header{
		ParentHash:  *(*[32]byte)(eh.ParentHash),
		UncleHash:   *(*[32]byte)(eh.UncleHash),
		Coinbase:    *(*[20]byte)(eh.FeeRecipient),
		Root:        *(*[32]byte)(eh.StateRoot),
		TxHash:      *(*[32]byte)(eh.TxHash),
		ReceiptHash: *(*[32]byte)(eh.ReceiptsRoot),
		Bloom:       *(*[256]byte)(eh.LogsBloom),
		Number:      new(big.Int).SetUint64(eh.BlockNumber),
		GasLimit:    eh.GasLimit,
		GasUsed:     eh.GasUsed,
		Time:        eh.Timestamp,
		Extra:       eh.ExtraData,
		MixDigest:   *(*[32]byte)(eh.MixDigest),
		Nonce:       *(*[8]byte)(eh.Nonce),
	}
	var difficulty big.Int
	difficulty.SetBytes(eh.PrevRandao)
	hdr.Difficulty = &difficulty

	var basefee big.Int
	if *(*[32]byte)(eh.BaseFeePerGas) != [32]byte{} {
		basefee.SetBytes(eh.BaseFeePerGas)
		hdr.BaseFee = &basefee
	}
	return &hdr
}

func (e *ExecutionPayload) EncodeRLP(w io.Writer) error {

	hdr := fillHdr(e.Header)
	var txs = make([]*types.Transaction, len(e.Transactions))
	for i, encTx := range e.Transactions {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(encTx); err != nil {
			return fmt.Errorf("invalid transaction %d: %v", i, err)
		}
		txs[i] = &tx
	}
	var uncles []*types.Header
	for i := 0; i < len(e.Uncles); i++ {
		uncles = append(uncles, fillHdr(e.Uncles[i]))

	}

	return rlp.Encode(w, extblock{
		Header: hdr,
		Txs:    txs,
		Uncles: uncles,
	})
}

func fillEHdr(h *types.Header) (*ExecutionHeader, error) {
	eh := &ExecutionHeader{}
	eh.ParentHash = h.ParentHash[:]
	eh.UncleHash = h.UncleHash[:]
	eh.FeeRecipient = h.Coinbase[:]
	eh.StateRoot = h.Root[:]
	eh.TxHash = h.TxHash[:]
	eh.ReceiptsRoot = h.ReceiptHash[:]
	eh.LogsBloom = h.Bloom[:]

	eh.PrevRandao = make([]byte, 32)
	h.Difficulty.FillBytes(eh.PrevRandao)

	eh.BlockNumber = h.Number.Uint64()
	eh.GasLimit = h.GasLimit
	eh.GasUsed = h.GasUsed
	eh.Timestamp = h.Time

	if len(h.Extra) > 32 {
		return nil, fmt.Errorf("invalid extradata length in block %d: %v", eh.BlockNumber, len(h.Extra))
	}
	eh.ExtraData = h.Extra

	eh.BaseFeePerGas = make([]byte, 32)
	if h.BaseFee != nil {
		h.BaseFee.FillBytes(eh.BaseFeePerGas)
	}
	eh.MixDigest = h.MixDigest[:]
	eh.Nonce = h.Nonce[:]
	//	e.BlockHash = make([]byte, 32)
	return eh, nil
}

func (e *ExecutionPayload) DecodeRLP(s *rlp.Stream) error {
	var eb extblock
	if err := s.Decode(&eb); err != nil {
		return err
	}

	eh, err := fillEHdr(eb.Header)
	if err != nil {
		return err
	}
	e.Header = eh

	for i := 0; i < len(eb.Txs); i++ {
		b, err := eb.Txs[i].MarshalBinary()
		if err != nil {
			return err
		}
		e.Transactions = append(e.Transactions, b)
	}
	for i := 0; i < len(eb.Uncles); i++ {
		eh, err := fillEHdr(eb.Uncles[i])
		if err != nil {
			return err
		}
		e.Uncles = append(e.Uncles, eh)
	}
	return nil
}
