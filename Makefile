BUILD_COMMANDS = ./cmd/converter

install:
	@go install $(BUILD_COMMANDS)

# requires sszgen on path (e.g. 'go install github.com/ferranbt/fastssz/sszgen')
sszgen:
	rm -f spec/spec_encoding.go
	~/go/bin/sszgen --path spec -objs Header,Block,BlockArchive,Receipt,Log
