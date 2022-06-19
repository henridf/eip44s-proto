# EIP-4444 Prototyping

This repo contains some early dev work around EIP-4444. Right now, the two main pieces are:

1. A SSZ-based specification for block archive files. These files contain block headers, bodies, and receipts. The specification is tracked in https://github.com/henridf/eip44s-proto/issues/1.
2. A command line tool `blart` (**bl**ock **ar**chive **t**ool) to encode/decode these files to/from RLP.


### blart: CLI tool
```sh
$ blart -h
Usage of blart:
  -f string
    	write data to given output file (default stdout)
  -hash
    	compute ssz hash of block list (read only mode, no output is written)
  -i string
    	format of input data [rlp,rlprc,ssz] (default "ssz")
  -info
    	print block number info (read only mode, no output is written)
  -o string
    	format for output data [rlp,rlprc,ssz], where rlp is the standard RLP block encoding and rlprc is rlp with interleaved receipts (default "ssz")
  -targetsize int
    	target output size (approximate) when encoding from rlp to ssz. Results in multiple sequential ssz files. Set '0' to slurp all data into one output file.
```

A note on the above formats: `rlp` is the existing rlp block format exported by geth. `rlprc` is like rlp, but with the addition of receipts (currently not in geth but in this fork: https://github.com/henridf/go-ethereum/commit/f50b363f78acd5ed0962f57164e60235db37cfe3).



An example that takes an input `rlprc`-format file, encodes it to ssz, then computes the hash tree root.

```sh 
$ blart -i rlprc -f out.ssz blocks-receipts-2000000-2100000.rlp

$ blart -info out.ssz 
Format version 0
First block: 2000000, last block: 2100001

$ blart -hash out.ssz 
hash_tree_root: 7eace3fd41367784d233117ef16f1c5828428b8502af8b7d3de317138777787b
```

#### Reading/writing multiple files

`blart`'s driving use case is to encode an entire chain history from rlp to ssz. Given that history (on most chains) is too large to fit in a single file, `blart` supports reading multiple input rlp/rlprc files, and outputting multiple ssz files. The input files are to be listed on the command line and should be contiguous and in order of increasing blocks. Presenting out-of-order and/or non-contiguous input files will result in an error. The `-targetsize` flag can be used to indicate the (approximate) desired size of output ssz files. When present, `blart` will write numbered output files with a naming scheme `name-0.ssz, name-1.ssz, ...`, where `name.ssz` is the parameter passed to the `-o` flag.

For example,

```sh
blart -i rlprc -o archive.ssz -targesize 100000000 blocks-receipts-0-999999.rlp blocks-receipts-1000000-1999999.rlp blocks-receipts-2000000-2999999.rlp blocks-receipts-3000000-3999999.rlp
```

will result in the four contiguous rlp block files being read, and written to files `archive-0.ssz, archive-1.ssz, ... archive-n.ssz` of size approximately 10MB. If the `-targetsize` parameter is absent, all input is read in and written to a single output file.


This size-based splitting is only supported when converting rlp to ssz. When converting ssz to rlp, if multiple input ssz files are provided, they are all read in and written to  a single rlp output.
