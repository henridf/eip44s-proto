BUILD_COMMANDS = ./cmd/bart

install:
	@go install $(BUILD_COMMANDS)

# requires sszgen on path (e.g. 'go install github.com/ferranbt/fastssz/sszgen')
sszgen:
	rm -f spec/spec_encoding.go
	~/go/bin/sszgen --path spec -objs Header,Block,ArchiveBody,ArchiveHeader,Receipt,Log

