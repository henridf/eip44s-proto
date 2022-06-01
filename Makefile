BUILD_COMMANDS = ./cmd/converter

install:
	@go install $(BUILD_COMMANDS)

# requires sszgen on path (e.g. 'go get github.com/ferranbt/fastssz/sszgen')
sszgen:
	rm -f spec/spec_encoding.go
	~/go/bin/sszgen --path spec -objs ExecutionHeader,ExecutionPayload,Blocks,ReceiptPayload,LogPayload
