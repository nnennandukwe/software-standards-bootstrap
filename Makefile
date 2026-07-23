.PHONY: test race vet build vuln verify

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

build:
	go build ./cmd/ssb

vuln:
	go tool govulncheck ./...

verify: test race vet build vuln
