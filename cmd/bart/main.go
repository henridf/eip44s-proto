package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/henridf/eip44s-proto/spec"
	"github.com/rs/zerolog"
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

func multiReader(filenames []string) (io.Reader, error) {
	var readers []io.Reader
	for i := 0; i < len(filenames); i++ {
		fh, err := os.Open(filenames[i])
		if err != nil {
			return nil, err
		}
		readers = append(readers, fh)
	}
	return io.MultiReader(readers...), nil
}

func numberedFileName(basename string, n int) string {
	suffix := filepath.Ext(basename)
	name := strings.TrimSuffix(basename, suffix)
	name = name + fmt.Sprintf("-%d", n) + suffix
	return name
}

func logger() zerolog.Logger {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	output.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s:", i)
	}
	return zerolog.New(output).With().Timestamp().Logger()
}

func main() {
	var ifmt string
	var ofmt = "ssz"
	var output string
	var hash bool
	var info bool
	var targetSize int

	flag.StringVar(&ofmt, "o", "ssz", "format for output data [rlp,rlprc,ssz], where rlp is the standard RLP block encoding and rlprc is rlp with interleaved receipts")
	flag.StringVar(&ifmt, "i", "ssz", "format of input data [rlp,rlprc,ssz]")
	flag.StringVar(&output, "f", "", "write data to given output file (default stdout)")
	flag.IntVar(&targetSize, "targetsize", 0, "target output size (approximate) when encoding from rlp to ssz. Results in multiple sequential ssz files. Set '0' to slurp all data into one output file.")
	flag.BoolVar(&hash, "hash", false, "compute ssz hash of block list (read only mode, no output is written)")
	flag.BoolVar(&info, "info", false, "print block number info (read only mode, no output is written)")

	flag.Parse()

	if ifmt != "rlprc" && ifmt != "rlp" && ifmt != "ssz" {
		usage(fmt.Errorf("invalid input format"))
	}
	if ofmt != "rlprc" && ofmt != "rlp" && ofmt != "ssz" {
		usage(fmt.Errorf("invalid output format"))
	}
	if !info && !hash && ifmt == ofmt {
		usage(fmt.Errorf("must provide different input and output formats"))
	}

	if (hash || info) && ifmt != "ssz" {
		usage(fmt.Errorf("-hash and -info require input ssz file"))
	}
	if targetSize != 0 && targetSize < 1000*1000 {
		usage(fmt.Errorf("-targetsize too small"))
	}

	args := flag.Args()
	if len(args) == 0 {
		usage(fmt.Errorf("must pass a file name with either rlp or ssz-encoded blocks"))
	}

	log := logger()

	if ifmt == "rlp" || ifmt == "rlprc" {
		mr, err := multiReader(args)
		if err != nil {
			bail(err)
		}
		reader := newChunkedRLPReader(mr, ifmt == "rlprc", targetSize, log)

		done := false
		exp := uint64(0)
		for i := 0; !done; i++ {
			arc, err := reader.readOneArchive()
			if err == io.EOF {
				done = true
			} else if err != nil {
				bail(fmt.Errorf("reading RLP: %s", err))
			}
			if exp > 0 && arc.Blocks[0].Header.BlockNumber != exp {
				bail(fmt.Errorf("Non-consecutive blocks (%d, expected %d)", arc.Blocks[0].Header.BlockNumber, exp))
			}
			exp = arc.Blocks[0].Header.BlockNumber + uint64(len(arc.Blocks))
			archdr := spec.ArchiveHeader{
				Version:         version,
				HeadBlockNumber: arc.Blocks[0].Header.BlockNumber,
				BlockCount:      uint32(len(arc.Blocks)),
			}
			filename := output
			if targetSize > 0 {
				filename = numberedFileName(output, i)
			}
			if err := writeSSZ(filename, arc, archdr); err != nil {
				bail(fmt.Errorf("writing SSZ: %s", err))
			}
		}
		os.Exit(0)
	}

	// ifmt == "ssz"
	if info {
		file, err := os.Open(args[0])
		if err != nil {
			bail(fmt.Errorf("opening file: %s", err))
		}
		archdr, err := readSSZHeader(file)
		if err != nil {
			bail(err)
		}

		fmt.Printf("Format version %d\n", archdr.Version)
		fmt.Printf("First block: %d, last block: %d\n", archdr.HeadBlockNumber, archdr.HeadBlockNumber+uint64(archdr.BlockCount)-1)
		os.Exit(0)
	}

	var arcs []spec.ArchiveBody
	for len(args) > 0 {
		var fn string
		fn, args = args[0], args[1:]

		file, err := os.Open(fn)
		if err != nil {
			bail(fmt.Errorf("opening file: %s", err))
		}

		log.Info().Str("name", fn).Msg("Reading SSZ archive file")
		archdr, err := readSSZHeader(file)
		if err != nil {
			bail(err)
		}

		arc, err := readSSZBlocks(file)
		if err != nil {
			bail(err)
		}
		if err := checkArchive(arc, archdr); err != nil {
			bail(fmt.Errorf("invalid archive: %s", err))
		}
		arcs = append(arcs, arc)

		if hash {
			h32, err := arc.HashTreeRoot()
			if err != nil {
				bail(fmt.Errorf("computing hash: %s", err))
			}
			fmt.Printf("hash_tree_root: %x\n", h32)
			os.Exit(0)
		}
	}
	log.Info().Str("name", output).Msg("Writing RLP file")
	if err := writeRLP(ofmt, output, arcs...); err != nil {
		bail(fmt.Errorf("writing RLP: %s", err))
	}
}

func checkArchive(arc spec.ArchiveBody, archdr spec.ArchiveHeader) error {
	if arc.Blocks[0].Header.BlockNumber != archdr.HeadBlockNumber {
		return fmt.Errorf("header has first block %d, but body has first block %d",
			archdr.HeadBlockNumber, arc.Blocks[0].Header.BlockNumber)
	}
	if len(arc.Blocks) != int(archdr.BlockCount) {
		return fmt.Errorf("header has block count %d, but body has %d blocks",
			archdr.BlockCount, len(arc.Blocks))
	}
	return nil
}

func writeSSZ(output string, arc spec.ArchiveBody, archdr spec.ArchiveHeader) error {
	var w io.Writer
	if output == "" {
		w = os.Stdout
	} else {
		fh, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			bail(fmt.Errorf("could not open output file %s: %s", output, err))
		}
		defer fh.Close()
		w = fh
	}
	b, err := archdr.MarshalSSZ()
	if err != nil {
		return fmt.Errorf("marshalling SSZ header: %s", err)
	}
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("writing SSZ header ssz: %s", err)
	}
	if b, err = arc.MarshalSSZ(); err != nil {
		return fmt.Errorf("marshalling SSZ body: %s", err)
	}
	if _, err = w.Write(b); err != nil {
		return fmt.Errorf("writing SSZ body: %s", err)
	}
	return nil
}

func writeRLP(ofmt string, output string, arcs ...spec.ArchiveBody) error {
	var w io.Writer
	if output == "" {
		w = os.Stdout
	} else {
		fh, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			bail(fmt.Errorf("could not open output file %s: %s", output, err))
		}
		defer fh.Close()
		w = fh
	}
	exp := uint64(0)
	for i := 0; i < len(arcs); i++ {
		arc := arcs[i]
		if exp > 0 && arc.Blocks[0].Header.BlockNumber != exp {
			bail(fmt.Errorf("Non-consecutive blocks (%d, expected %d)", arc.Blocks[0].Header.BlockNumber, exp))
		}
		exp = arc.Blocks[len(arc.Blocks)-1].Header.BlockNumber + 1
		if err := writeArcRLP(w, arc, ofmt == "rlprc"); err != nil {
			bail(fmt.Errorf("writing RLP: %s", err))
		}
	}
	return nil
}

func writeArcRLP(w io.Writer, arc spec.ArchiveBody, receipts bool) error {
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

type countingReader struct {
	r io.Reader
	n int
}

func (cr *countingReader) Read(p []byte) (n int, err error) {
	n, err = cr.r.Read(p)
	cr.n += n
	return n, err
}

type chunkedRLPReader struct {
	stream     *rlp.Stream
	receipts   bool
	targetSize int
	cr         *countingReader
	log        zerolog.Logger
}

func newChunkedRLPReader(r io.Reader, receipts bool, targetSize int, log zerolog.Logger) *chunkedRLPReader {
	cr := &countingReader{r: r}
	stream := rlp.NewStream(cr, 0)
	return &chunkedRLPReader{
		stream,
		receipts,
		targetSize,
		cr,
		log,
	}
}

func (c *chunkedRLPReader) readOneArchive() (spec.ArchiveBody, error) {
	var blocks []*spec.Block
	// xxx not checking maxblocks
	var err error
	for i := 0; true; i++ {
		_, _, err = c.stream.Kind()
		if err == io.EOF {
			c.log.Info().Int("size (bytes)", c.cr.n).Msg("Read final archive")
			break
		}
		if c.targetSize > 0 && c.cr.n >= c.targetSize {
			c.log.Info().Int("size (bytes)", c.cr.n).Msg("Read one archive")
			break
		}
		if err != nil {
			break
		}

		var b spec.Block
		if c.receipts {
			err = c.stream.Decode(&b)
			if err != nil && err != io.EOF {
				return spec.ArchiveBody{}, fmt.Errorf("decoding RLP block %d: %v", i, err)
			}
		} else {
			var bn spec.BlockNoReceipts
			err = c.stream.Decode(&bn)
			if err != nil && err != io.EOF {
				return spec.ArchiveBody{}, fmt.Errorf("decoding RLP block %d: %v", i, err)
			}
			b = (spec.Block)(bn)
		}
		blocks = append(blocks, &b)
	}
	c.cr.n = 0
	return spec.ArchiveBody{
		Blocks: blocks,
	}, err
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
