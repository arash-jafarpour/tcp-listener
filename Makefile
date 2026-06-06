build:
	@go build -o tcp-listener

run: build
	@./tcp-listener

run-verbose: build
	@./tcp-listener --verbose

run-hex: build
	@./tcp-listener --verbose --dump hex

run-hexdump: build
	@./tcp-listener --verbose --dump hexdump
