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
	var ifmt string
	var ofmt = "ssz"
	var output string

	flag.StringVar(&ofmt, "o", "ssz", "format for output data [rlp,rlprec,ssz], where rlp is the standard RLP block encoding and rlprc is rlp with interleaved receipts")
	flag.StringVar(&ifmt, "i", "", "format of input data [rlp,rlprec,ssz]")
	flag.StringVar(&output, "f", "", "write data to given output file (default stdout)")

	flag.Parse()

	if ifmt == "" {
		bail(fmt.Errorf("-i must be provided"))
	}
	if ifmt == ofmt {
		bail(fmt.Errorf("must provide different input and output formats"))
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
	var arc spec.BlockArchive
	if ifmt == "rlp" || ifmt == "rlprc" {
		var err error
		arc, err = readRLP(args[0], ifmt == "rlprc")
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading RLP: %s\n", err)
			os.Exit(1)
		}
	}
	if ifmt == "ssz" {
		var err error
		arc, err = readSSZ(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading SSZ: %s\n", err)
			os.Exit(1)
		}
	}

	if ofmt == "ssz" {
		if err := writeSSZ(w, arc); err != nil {
			fmt.Fprintf(os.Stderr, "writing SSZ: %s\n", err)
			os.Exit(1)
		}
		return
	}
	if ofmt == "rlp" || ofmt == "rlprc" {
		if err := writeRLP(w, arc, ofmt == "rlprc"); err != nil {
			fmt.Fprintf(os.Stderr, "writing RLP: %s\n", err)
			os.Exit(1)
		}
	}
}

func readRLP(path string, receipts bool) (spec.BlockArchive, error) {
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

		var b spec.Block
		if receipts {
			err = stream.Decode(&b)
			if err != nil {
				return spec.BlockArchive{}, fmt.Errorf("decoding RLP block %d: %v", i, err)
			}
		} else {
			var bn spec.BlockNoReceipts
			err = stream.Decode(&bn)
			if err != nil {
				return spec.BlockArchive{}, fmt.Errorf("decoding RLP block %d: %v", i, err)
			}
			b = (spec.Block)(bn)
		}
		blocks = append(blocks, &b)
	}

	return spec.BlockArchive{
		Blocks: blocks,
	}, nil
}

func writeRLP(w io.Writer, arc spec.BlockArchive, receipts bool) error {
	for i := 0; i < len(arc.Blocks); i++ {
		var err error
		if receipts {
			err = rlp.Encode(w, arc.Blocks[i])
		} else {
			err = rlp.Encode(w, (*spec.BlockNoReceipts)(arc.Blocks[i]))
		}
		if err != nil {
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
