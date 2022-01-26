package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/henridf/eip4444/spec"
)

func bail(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	var encode bool
	var decode bool

	// xxx toSSZ/toRLP ?
	flag.BoolVar(&encode, "encode", false, "encode input rlp into ssz")
	flag.BoolVar(&decode, "decode", false, "decode input ssz into rlp")

	flag.Parse()

	if decode == encode {
		bail(fmt.Errorf("must select either -encode or -decode"))
	}

	args := flag.Args()
	if len(args) == 0 {
		bail(fmt.Errorf("must pass a file name with either rlp or ssz-encoded blocks"))
	}

	if encode {
		blocks, err := readRLP(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading RLP: %s\n", err)
			os.Exit(1)
		}
		if err := writeSSZ("out.ssz", blocks); err != nil {
			fmt.Fprintf(os.Stderr, "writing  RLP: %s\n", err)
			os.Exit(1)
		}
		return
	}
	if decode {
		blocks, err := readSSZ(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading SSZ: %s\n", err)
			os.Exit(1)
		}
		if err := writeRLP("out.rlp", blocks); err != nil {
			fmt.Fprintf(os.Stderr, "reading RLP: %s\n", err)
			os.Exit(1)
		}
	}
}

func readRLP(path string) (spec.Blocks, error) {
	fh, err := os.Open(path)
	if err != nil {
		return spec.Blocks{}, err
	}
	defer fh.Close()

	stream := rlp.NewStream(fh, 0)
	var rlpBlocks [][]byte
	// xxx not checking maxblocks
	for i := 0; i < spec.MaxBlocks; i++ {
		kind, size, err := stream.Kind()
		if err == io.EOF {
			break
		}
		if err != nil {
			return spec.Blocks{}, fmt.Errorf("reading kind %d: %v", i, err)
		}
		fmt.Printf("Decoding %s of size %d\n", kind, size)
		var r rlp.RawValue
		err = stream.Decode(&r)
		if err != nil {
			return spec.Blocks{}, fmt.Errorf("decoding RLP block %d: %v", i, err)
		}
		// xxx not checking spec.MaxBlockSize
		rlpBlocks = append(rlpBlocks, r)
	}

	for i := 0; i < len(rlpBlocks); i++ {
		fmt.Printf("block %d: len %d\n", i, len(rlpBlocks[i]))
	}

	return spec.Blocks{
		RlpPayload: rlpBlocks,
	}, nil
}

func writeRLP(path string, blocks spec.Blocks) error {
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()

	for i := 0; i < len(blocks.RlpPayload); i++ {
		if _, err := fh.Write(blocks.RlpPayload[i]); err != nil {
			return fmt.Errorf("writing RLP blocks: %s\n", err)
		}
	}
	return nil
}

func writeSSZ(path string, blocks spec.Blocks) error {
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()

	b, err := blocks.MarshalSSZ()
	if err != nil {
		return fmt.Errorf("Marshalling: %s\n", err)
	}
	if _, err := fh.Write(b); err != nil {
		return fmt.Errorf("Writing ssz: %s\n", err)
	}
	return nil
}
func readSSZ(path string) (spec.Blocks, error) {
	blocks := spec.Blocks{}
	fh, err := os.Open(path)
	if err != nil {
		return blocks, err
	}
	b, err := ioutil.ReadAll(fh)
	if err != nil {
		return blocks, fmt.Errorf("reading ssz file: %s\n", err)
	}
	if err = blocks.UnmarshalSSZ(b); err != nil {
		return blocks, fmt.Errorf("unmarshalling ssz: %s\n", err)
	}
	return blocks, nil
}
