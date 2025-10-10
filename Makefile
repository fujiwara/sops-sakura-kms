.PHONY: clean test

sops-sakura-kms: go.* *.go
	go build -o $@ ./cmd/sops-sakura-kms

clean:
	rm -rf sops-sakura-kms dist/

test:
	go test -v ./...

install:
	go install github.com/fujiwara/sops-sakura-kms/cmd/sops-sakura-kms

dist:
	goreleaser build --snapshot --clean
