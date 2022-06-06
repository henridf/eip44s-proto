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

func readRLP(path string) (spec.BlockArchive, error) {
	fh, err := os.Open(path)
	if err != nil {
		return spec.BlockArchive{}, err
	}
	defer fh.Close()

	stream := rlp.NewStream(fh, 0)
	var blocks []*spec.Block
	// xxx not checking maxblocks
	for i := 0; i < spec.MaxBlocks; i++ {
		_, _, err := stream.Kind()
		if err == io.EOF {
			break
		}
		if err != nil {
			return spec.BlockArchive{}, fmt.Errorf("reading kind %d: %v", i, err)
		}
		var e spec.Block
		err = stream.Decode(&e)
		if err != nil {
			return spec.BlockArchive{}, fmt.Errorf("decoding RLP block %d: %v", i, err)
		}
		// xxx not checking spec.MaxBlockSize
		blocks = append(blocks, &e)
	}

	return spec.BlockArchive{
		Blocks: blocks,
	}, nil
}

func writeRLP(w io.Writer, arc spec.BlockArchive) error {
	for i := 0; i < len(arc.Blocks); i++ {
		if err := rlp.Encode(w, (*spec.Block)(arc.Blocks[i])); err != nil {
			return fmt.Errorf("writing RLP-encoded block: %s\n", err)
		}
	}
	return nil
}

func writeSSZ(w io.Writer, blocks spec.BlockArchive) error {
	b, err := blocks.MarshalSSZ()
	if err != nil {
		return fmt.Errorf("Marshalling: %s\n", err)
	}
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("Writing ssz: %s\n", err)
	}
	return nil
}
func readSSZ(path string) (spec.BlockArchive, error) {
	blocks := spec.BlockArchive{}
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
