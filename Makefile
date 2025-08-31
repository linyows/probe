CERT_DIR := testdata/certs

default: build

build:
	env CGO_ENABLED=0 go build -ldflags="-s -w" -o probe ./cmd/probe/...

test:
	@go test -v ./... -coverprofile=coverage.out -covermode=count | \
		grep -v '^=== RUN' | \
		sed -E 's/--- PASS:/\x1B[38;5;34m✔︎\x1B[0m/g' | \
		sed -E 's/--- FAIL:/\x1B[31m✘\x1B[0m/g' | \
		sed -E 's/^PASS$$/\x1B[38;5;34m✔︎ Pass\x1B[0m/' | \
		sed -E 's/^FAIL$$/\x1B[31m✘ Fail\x1B[0m/'

lint:
	golangci-lint run ./...

key:
	@rm -rf keys/*.pem
	@mkdir keys
	@openssl req -x509 -days 10 -newkey ED25519 -nodes -out ./keys/cert.pem -keyout ./keys/key.pem -subj "/C=/ST=/L=/O=/OU=/CN=example.local" &>/dev/null

code:
	#@which buf || brew install bufbuild/buf/buf
	buf generate

http_server:
	go run github.com/mccutchen/go-httpbin/v2/cmd/go-httpbin@latest -host 127.0.0.1 -port 8080

http_server_tls:
	go run github.com/mccutchen/go-httpbin/v2/cmd/go-httpbin@latest -host 127.0.0.1 -port 8080 \
		-https-cert-file ./testdata/server.crt \
		-https-key-file ./testdata/server.key

grpc_server:
	go run grpc/testserver/*.go

grpc_server_tls:
	go run grpc/testserver/*.go \
		-tls \
		-cert="./testdata/certs/server.crt" \
		-key="./testdata/certs/server.key" \
		-port=50052

gen_server_keys:
	testdata/gen.sh

gen_grpc_server:
	@cd grpc/testserver && \
		protoc --go_out=. --go-grpc_out=. ./pb/user_service.proto
