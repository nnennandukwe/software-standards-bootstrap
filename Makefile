.PHONY: test race vet build vuln verify verify-release-archives

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

verify-release-archives:
	go run github.com/goreleaser/goreleaser/v2@v2.17.0 release --snapshot --clean --skip=sbom
	SSB_RELEASE_ARCHIVE_DIR=dist SSB_RELEASE_SOURCE_REF=HEAD go test ./internal/releaseconfig -run '^TestGeneratedReleaseArchivesContainCompleteSkill$$'
