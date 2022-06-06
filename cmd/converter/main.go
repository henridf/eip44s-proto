package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/henridf/eip44s-proto/spec"
)

func bail(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	var encode bool
	var decode bool
	var output string

	// xxx toSSZ/toRLP ?
	flag.BoolVar(&encode, "encode", false, "encode input rlp into ssz")
	flag.BoolVar(&decode, "decode", false, "decode input ssz into rlp")
	flag.StringVar(&output, "o", "", "write data to output file")

	flag.Parse()

	if decode == encode {
		bail(fmt.Errorf("must select either -encode or -decode"))
	}

	args := flag.Args()
	if len(args) == 0 {
		bail(fmt.Errorf("must pass a file name with either rlp or ssz-encoded blocks"))
	}

	var w io.Writer
	if output == "" {
		w = os.Stdout
	} else {
		fh, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not open output file %s: %s\n", output, err)
			os.Exit(1)
		}
		defer fh.Close()
		w = fh
	}
	if encode {
		blocks, err := readRLP(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading RLP: %s\n", err)
			os.Exit(1)
		}
		if err := writeSSZ(w, blocks); err != nil {
			fmt.Fprintf(os.Stderr, "writing SSZ: %s\n", err)
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
		if err := writeRLP(w, blocks); err != nil {
			fmt.Fprintf(os.Stderr, "writing RLP: %s\n", err)
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
	var blocks []*spec.ExecutionPayload
	// xxx not checking maxblocks
	for i := 0; i < spec.MaxBlocks; i++ {
		_, _, err := stream.Kind()
		if err == io.EOF {
			break
		}
		if err != nil {
			return spec.Blocks{}, fmt.Errorf("reading kind %d: %v", i, err)
		}
		var e spec.ExecutionPayload
		err = stream.Decode(&e)
		if err != nil {
			return spec.Blocks{}, fmt.Errorf("decoding RLP block %d: %v", i, err)
		}
		// xxx not checking spec.MaxBlockSize
		blocks = append(blocks, &e)
	}

	return spec.Blocks{
		Payload: blocks,
	}, nil
}

func writeRLP(w io.Writer, blocks spec.Blocks) error {
	for i := 0; i < len(blocks.Payload); i++ {
		if err := rlp.Encode(w, (*spec.ExecutionPayload)(blocks.Payload[i])); err != nil {
			return fmt.Errorf("writing RLP-encoded block: %s\n", err)
		}
	}
	return nil
}

func writeSSZ(w io.Writer, blocks spec.Blocks) error {
	b, err := blocks.MarshalSSZ()
	if err != nil {
		return fmt.Errorf("Marshalling: %s\n", err)
	}
	if _, err := w.Write(b); err != nil {
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
