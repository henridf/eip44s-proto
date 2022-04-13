BUILD_COMMANDS = ./cmd/converter

install:
	@go install $(BUILD_COMMANDS)

# need to 'go get github.com/ferranbt/fastssz/sszgen'
sszgen:
	rm -f spec/spec_encoding.go
	sszgen --path spec
