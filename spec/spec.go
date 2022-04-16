package spec

// go run sszgen/*.go --path ../../work/eip4444/

const MaxBlocks = 2000000
const MaxBlockSize = 268435456

type Blocks struct {
	RlpPayload [][]byte `ssz-max:"2000000,268435456"`
}
