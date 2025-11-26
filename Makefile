.PHONY: clean test

IMAGE_VERSION ?= $(shell git describe --tags --abbrev)

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

image-push: clean
	go mod vendor
	docker buildx build --platform linux/amd64,linux/arm64 \
	-t ghcr.io/fujiwara/sops-sakura-kms:$(IMAGE_VERSION) \
	--push \
	-f Dockerfile .

image-load: clean
	go mod vendor
	docker buildx build --platform linux/amd64 \
	-t ghcr.io/fujiwara/sops-sakura-kms:$(IMAGE_VERSION) \
	--load \
	-f Dockerfile .
