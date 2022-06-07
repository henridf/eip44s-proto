package spec

import (
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// from core/types/block.go
type extblock struct {
	Header *types.Header
	Txs    []*types.Transaction
	Uncles []*types.Header
}

func fillHdr(eh *Header) *types.Header {
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
	difficulty.SetBytes(eh.Difficulty)
	hdr.Difficulty = &difficulty

	var basefee big.Int
	if *(*[32]byte)(eh.BaseFeePerGas) != [32]byte{} {
		basefee.SetBytes(eh.BaseFeePerGas)
		hdr.BaseFee = &basefee
	}
	return &hdr
}

func (r *Receipt) EncodeRLP(w io.Writer) error {
	buf := rlp.NewEncoderBuffer(w)
	outerList := buf.List()
	if len(r.PostState) == 0 {
		if r.Status == types.ReceiptStatusFailed {
			buf.WriteBytes([]byte{})
		} else {
			buf.WriteBytes([]byte{0x01})
		}
	} else {
		buf.WriteBytes(r.PostState)
	}

	buf.WriteUint64(r.CumulativeGasUsed)
	logList := buf.List()
	for _, log := range r.Logs {
		if err := rlp.Encode(buf, log); err != nil {
			return err
		}
	}
	buf.ListEnd(logList)
	buf.ListEnd(outerList)
	return buf.Flush()
}

func blockEncodeRLP(e *Block, w io.Writer, receipts bool) error {

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

	err := rlp.Encode(w, extblock{
		Header: hdr,
		Txs:    txs,
		Uncles: uncles,
	})
	if !receipts || err != nil {
		return err
	}
	return rlp.Encode(w, e.Receipts)
}

func (e *Block) EncodeRLP(w io.Writer) error {
	return blockEncodeRLP(e, w, true)
}

type BlockNoReceipts Block

func (e *BlockNoReceipts) EncodeRLP(w io.Writer) error {
	return blockEncodeRLP((*Block)(e), w, false)
}

func fillEHdr(h *types.Header) (*Header, error) {
	eh := &Header{}
	eh.ParentHash = h.ParentHash[:]
	eh.UncleHash = h.UncleHash[:]
	eh.FeeRecipient = h.Coinbase[:]
	eh.StateRoot = h.Root[:]
	eh.TxHash = h.TxHash[:]
	eh.ReceiptsRoot = h.ReceiptHash[:]
	eh.LogsBloom = h.Bloom[:]

	eh.Difficulty = make([]byte, 32)
	h.Difficulty.FillBytes(eh.Difficulty)

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

func (e *Block) DecodeRLP(s *rlp.Stream) error {
	return blockDecodeRLP(e, s, true)
}

func (e *BlockNoReceipts) DecodeRLP(s *rlp.Stream) error {
	return blockDecodeRLP((*Block)(e), s, false)
}

func blockDecodeRLP(e *Block, s *rlp.Stream, withreceipts bool) error {
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
	if !withreceipts {
		return nil
	}
	var receipts []*types.ReceiptForStorage
	if err := s.Decode(&receipts); err != nil {
		return err
	}
	for i := 0; i < len(receipts); i++ {
		p := &Receipt{}
		if len(receipts[i].PostState) > 0 {
			p.PostState = receipts[i].PostState
		} else {
			p.Status = receipts[i].Status
		}
		p.CumulativeGasUsed = receipts[i].CumulativeGasUsed
		for _, rlplog := range receipts[i].Logs {
			log := &Log{Address: rlplog.Address[:], Data: rlplog.Data}
			for j := 0; j < len(rlplog.Topics); j++ {
				topic := rlplog.Topics[j]
				// xxx ugly conversion from []common.Hash to [][]byte...
				// maybe just common.Hash directly? (here and elsewhere)
				log.Topics = append(log.Topics, []byte(topic[:]))
			}
			p.Logs = append(p.Logs, log)
		}
		e.Receipts = append(e.Receipts, p)
	}

	return nil
}
