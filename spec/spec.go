package spec

// go run sszgen/*.go --path ../../work/eip4444/

const MaxBlocks = 2000000

type Blocks struct {
	HeadBlockNumber uint64
	BlockCount      uint32
	Payload         []*ExecutionPayload `ssz-max:"2000000"`
}

type ExecutionHeader struct {
	ParentHash    []byte `ssz-size:"32"`
	UncleHash     []byte `ssz-size:"32"`
	FeeRecipient  []byte `ssz-size:"20"` // 84
	StateRoot     []byte `ssz-size:"32"`
	TxHash        []byte `ssz-size:"32"`
	ReceiptsRoot  []byte `ssz-size:"32"`  // 180
	LogsBloom     []byte `ssz-size:"256"` // 436
	PrevRandao    []byte `ssz-size:"32"`  // should we just call this Difficulty since it's pre-merge blocks?
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

type ExecutionPayload struct {
	Header       *ExecutionHeader   `ssz-max:"604"`
	Transactions [][]byte           `ssz-max:"1048576,1073741824" ssz-size:"?,?"`
	Uncles       []*ExecutionHeader `ssz-max:"6040"`
}

type ReceiptPayload struct {
	ReceiptType       uint8
	PostState         []byte `ssz-size:"32"`
	Status            uint64
	CumulativeGasUsed uint64
	Bloom             []byte        `ssz-size:"256"`
	Logs              []*LogPayload `ssz-max:"4194533"`
	TxHash            []byte        `ssz-size:"32"`
	ContractAddress   []byte        `ssz-size:"20"`
	GasUsed           uint64
	BlockHash         []byte `ssz-size:"32"`
	BlockNumber       uint64
	TxIndex           uint32
}

type LogPayload struct {
	address      []byte   `ssz-size:"20"`
	topics       [][]byte `ssz-max:"4" ssz-size:"?,32"` // 148
	data         []byte   `ssz-size:"4194304"`          // 4194452
	block_number uint64   // 4194458
	tx_hash      []byte   `ssz-size:"32"` // 4194490
	tx_index     uint32
	block_hash   []byte `ssz-size:"32"` // 4194526
	index        uint64 // 4194532
	removed      bool
}
