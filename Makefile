build:
	@go build -o tcp-listener
	
build-nocgo:
	@CGO_ENABLED=0 go build -o tcp-listener-nocgo .

run: build
	@./tcp-listener

run-verbose: build
	@./tcp-listener --verbose

run-hex: build
	@./tcp-listener --verbose --dump hex

run-hexdump: build
	@./tcp-listener --verbose --dump hexdump
