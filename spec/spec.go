package spec

// go run sszgen/*.go --path ../../work/eip4444/

const MaxBlocks = 1000
const MaxBlockSize = 268435456

type Blocks struct {
	RlpPayload [][]byte `ssz-max:"100,268435456"`
}
