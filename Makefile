build:
	@go build -o tcp-listener
	
build-nocgo:
	@CGO_ENABLED=0 go build -o tcp-listener-nocgo .

run: build
	@./tcp-listener listen

run-verbose: build
	@./tcp-listener listen --verbose

run-hex: build
	@./tcp-listener listen --verbose --dump hex

run-hexdump: build
	@./tcp-listener listen --verbose --dump hexdump
