default: build

build:
	env CGO_ENABLED=0 go build -ldflags="-s -w" -o probe ./cmd/probe/...

test:
	go test ./...

key:
	@rm -rf keys/*.pem
	@mkdir keys
	@openssl req -x509 -days 10 -newkey ED25519 -nodes -out ./keys/cert.pem -keyout ./keys/key.pem -subj "/C=/ST=/L=/O=/OU=/CN=example.local" &>/dev/null

code:
	#@which buf || brew install bufbuild/buf/buf
	buf generate
