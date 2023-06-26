all: build

clean:
	@rm -f bin/*

build:
	@go mod tidy
	@cd cmd/server && go build -o ../../bin/modem_prod_server
