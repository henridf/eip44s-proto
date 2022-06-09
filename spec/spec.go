package spec

// go run sszgen/*.go --path ../../work/eip4444/

const MaxBlocks = 1000000

type ArchiveHeader struct {
	Version         uint64
	HeadBlockNumber uint64
	BlockCount      uint32
}
type ArchiveBody struct {
	Blocks []*Block `ssz-max:"1000000"`
}

type Block struct {
	Header       *Header    `ssz-max:"604"`
	Transactions [][]byte   `ssz-max:"1048576,1073741824" ssz-size:"?,?"`
	Uncles       []*Header  `ssz-max:"6040"`
	Receipts     []*Receipt `ssz-max:"4194452"`
}

type Header struct {
	ParentHash    []byte `ssz-size:"32"`
	UncleHash     []byte `ssz-size:"32"`
	FeeRecipient  []byte `ssz-size:"20"` // 84
	StateRoot     []byte `ssz-size:"32"`
	TxHash        []byte `ssz-size:"32"`
	ReceiptsRoot  []byte `ssz-size:"32"`  // 180
	LogsBloom     []byte `ssz-size:"256"` // 436
	Difficulty    []byte `ssz-size:"32"`
	BlockNumber   uint64
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64 // 500
	ExtraData     []byte `ssz-max:"32"`
	BaseFeePerGas []byte `ssz-size:"32"`
	MixDigest     []byte `ssz-size:"32"`
	Nonce         []byte `ssz-size:"8"` // 604
	//	BlockHash     []byte   `ssz-size:"32"`
}

type Receipt struct {
	PostState         []byte `ssz-max:"32"`
	Status            uint64
	CumulativeGasUsed uint64
	Logs              []*Log `ssz-max:"4194452"` // xxx
}

type Log struct {
	Address []byte   `ssz-size:"20"`
	Topics  [][]byte `ssz-max:"4" ssz-size:"?,32"` // 148
	Data    []byte   `ssz-max:"4194304"`           // 4194452
}
