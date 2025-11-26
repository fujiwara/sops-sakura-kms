FROM --platform=${BUILDPLATFORM} golang:1.25 AS build-env

ARG TARGETOS
ARG TARGETARCH

ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}
ENV CGO_ENABLED=0
RUN mkdir -p /go/src/github.com/fujiwara/sops-sakura-kms
COPY . /go/src/github.com/fujiwara/sops-sakura-kms
WORKDIR /go/src/github.com/fujiwara/sops-sakura-kms
RUN make clean && make

FROM --platform=${BUILDPLATFORM} ghcr.io/getsops/sops:v3.11.0 AS sops

FROM gcr.io/distroless/static-debian12
LABEL maintainer="fujiwara <fujiwara.shunichiro@gmail.com>"

COPY --from=build-env /go/src/github.com/fujiwara/sops-sakura-kms/sops-sakura-kms /usr/local/bin/sops-sakura-kms
COPY --from=sops /usr/local/bin/sops /usr/local/bin/sops
ENTRYPOINT ["/usr/local/bin/sops-sakura-kms"]
