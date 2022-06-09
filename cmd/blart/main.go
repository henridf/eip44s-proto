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

const version = 0

func bail(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	os.Exit(1)
}

func usage(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	var ifmt string
	var ofmt = "ssz"
	var output string
	var hash bool
	var info bool

	flag.StringVar(&ofmt, "o", "ssz", "format for output data [rlp,rlprec,ssz], where rlp is the standard RLP block encoding and rlprc is rlp with interleaved receipts")
	flag.StringVar(&ifmt, "i", "ssz", "format of input data [rlp,rlprec,ssz]")
	flag.StringVar(&output, "f", "", "write data to given output file (default stdout)")
	flag.BoolVar(&hash, "hash", false, "compute ssz hash of block list (read only mode, no output is written)")
	flag.BoolVar(&info, "info", false, "print block number info (read only mode, no output is written)")

	flag.Parse()

	if ifmt == "" {
		usage(fmt.Errorf("-i must be provided"))
	}
	if !info && !hash && ifmt == ofmt {
		usage(fmt.Errorf("must provide different input and output formats"))
	}

	if (hash || info) && ifmt != "ssz" {
		usage(fmt.Errorf("-hash and -info require input ssz file"))
	}

	args := flag.Args()
	if len(args) == 0 {
		usage(fmt.Errorf("must pass a file name with either rlp or ssz-encoded blocks"))
	}

	var archdr spec.ArchiveHeader
	var arc spec.ArchiveBody
	_, _ = arc.HashTreeRoot()
	if ifmt == "rlp" || ifmt == "rlprc" {
		var err error
		arc, err = readRLP(args[0], ifmt == "rlprc")
		if err != nil {
			bail(fmt.Errorf("reading RLP: %s", err))
		}
		archdr = spec.ArchiveHeader{
			Version:         version,
			HeadBlockNumber: arc.Blocks[0].Header.BlockNumber,
			BlockCount:      uint32(len(arc.Blocks)),
		}
	}
	if ifmt == "ssz" {
		var err error

		file, err := os.Open(args[0])
		if err != nil {
			bail(fmt.Errorf("opening file: %s", err))
		}
		defer file.Close()

		archdr, err = readSSZHeader(file)
		if err != nil {
			bail(err)
		}

		if info {
			fmt.Printf("Format version %d\n", archdr.Version)
			fmt.Printf("First block: %d, last block: %d\n", archdr.HeadBlockNumber, archdr.HeadBlockNumber+uint64(archdr.BlockCount))
			os.Exit(0)
		}

		arc, err = readSSZBlocks(file)
		if err != nil {
			bail(err)
		}

		if arc.Blocks[0].Header.BlockNumber != archdr.HeadBlockNumber {
			bail(fmt.Errorf("invalid archive: header has first block %d, but body has first block %d",
				archdr.HeadBlockNumber, arc.Blocks[0].Header.BlockNumber))
		}
		if len(arc.Blocks) != int(archdr.BlockCount) {
			bail(fmt.Errorf("invalid archive: header has block count %d, but body has %d blocks",
				archdr.BlockCount, len(arc.Blocks)))
		}
	}

	if hash {
		h32, err := arc.HashTreeRoot()
		if err != nil {
			bail(fmt.Errorf("computing hash: %s", err))
		}
		fmt.Printf("hash_tree_root: %x\n", h32)
		os.Exit(0)
	}

	var writer io.Writer
	if output == "" {
		writer = os.Stdout
	} else {
		fh, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			bail(fmt.Errorf("could not open output file %s: %s", output, err))
		}
		defer fh.Close()
		writer = fh
	}

	if ofmt == "ssz" {
		if err := writeSSZ(writer, archdr, arc); err != nil {
			bail(fmt.Errorf("writing SSZ: %s", err))
		}
		return
	}
	if ofmt == "rlp" || ofmt == "rlprc" {
		if err := writeRLP(writer, arc, ofmt == "rlprc"); err != nil {
			bail(fmt.Errorf("writing RLP: %s", err))
		}
	}
}

func readRLP(path string, receipts bool) (spec.ArchiveBody, error) {
	fh, err := os.Open(path)
	if err != nil {
		return spec.ArchiveBody{}, err
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
			return spec.ArchiveBody{}, fmt.Errorf("reading kind %d: %v", i, err)
		}

		var b spec.Block
		if receipts {
			err = stream.Decode(&b)
			if err != nil {
				return spec.ArchiveBody{}, fmt.Errorf("decoding RLP block %d: %v", i, err)
			}
		} else {
			var bn spec.BlockNoReceipts
			err = stream.Decode(&bn)
			if err != nil {
				return spec.ArchiveBody{}, fmt.Errorf("decoding RLP block %d: %v", i, err)
			}
			b = (spec.Block)(bn)
		}
		blocks = append(blocks, &b)
	}

	return spec.ArchiveBody{
		Blocks: blocks,
	}, nil
}

func writeRLP(w io.Writer, arc spec.ArchiveBody, receipts bool) error {
	for i := 0; i < len(arc.Blocks); i++ {
		var err error
		if receipts {
			err = rlp.Encode(w, arc.Blocks[i])
		} else {
			err = rlp.Encode(w, (*spec.BlockNoReceipts)(arc.Blocks[i]))
		}
		if err != nil {
			return fmt.Errorf("writing RLP-encoded block: %s", err)
		}
	}
	return nil
}

func writeSSZ(w io.Writer, hdr spec.ArchiveHeader, blocks spec.ArchiveBody) error {
	b, err := hdr.MarshalSSZ()
	if err != nil {
		return fmt.Errorf("marshalling header: %s", err)
	}
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("writing header ssz: %s", err)
	}

	b, err = blocks.MarshalSSZ()
	if err != nil {
		return fmt.Errorf("marshalling body: %s", err)
	}
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("writing body ssz: %s", err)
	}
	return nil
}

func readSSZHeader(r io.Reader) (spec.ArchiveHeader, error) {
	var h spec.ArchiveHeader
	sz := h.SizeSSZ()
	buf := make([]byte, sz)

	if _, err := io.ReadFull(r, buf); err != nil {
		return spec.ArchiveHeader{}, err
	}
	if err := h.UnmarshalSSZ(buf); err != nil {
		return spec.ArchiveHeader{}, fmt.Errorf("unmarshalling ssz: %s", err)
	}
	return h, nil
}

func readSSZBlocks(r io.Reader) (spec.ArchiveBody, error) {
	var blocks spec.ArchiveBody
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return blocks, fmt.Errorf("reading ssz file: %s", err)
	}
	if err = blocks.UnmarshalSSZ(b); err != nil {
		return blocks, fmt.Errorf("unmarshalling ssz: %s", err)
	}
	return blocks, nil
}
