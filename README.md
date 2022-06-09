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
        format of input data [rlp,rlprec,ssz] (default "ssz")
  -info
        print block number info (read only mode, no output is written)
  -o string
        format for output data [rlp,rlprec,ssz], where rlp is the standard RLP block encoding and rlprc is rlp with interleaved receipts (default "ssz")
```

A note on the above formats: `rlp` is the existing rlp block format exported by geth. `rlprec` is like rlp, but with the addition of receipts (currently not in geth but in this fork: https://github.com/henridf/go-ethereum/commit/f50b363f78acd5ed0962f57164e60235db37cfe3).



An example that takes an input `rlprec`-file, encodes it to ssz, then computes the hash tree root.

```sh 
$ blart -i rlprc -f out.ssz blocks-receipts-2000000-2100000.rlp

$ blart -info out.ssz 
Format version 0
First block: 2000000, last block: 2100001

$ blart -hash out.ssz 
hash_tree_root: 7eace3fd41367784d233117ef16f1c5828428b8502af8b7d3de317138777787b
```
