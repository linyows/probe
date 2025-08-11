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
