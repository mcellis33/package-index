#!/bin/bash -e
#
# This script builds and tests the repo. If this script succeeds, the
# resulting package-index docker image is shippable.
#
# Steps:
#
# 1. Build package-index and test-suite Go binaries and dockerize them
# 2. Run unit tests with coverage
# 3. Run test-suite against package-index image
# 4. Run benchmarks with memory stats
#
# Notes:
#
# Each step runs in docker to isolate its environment. The Go version is
# pinned to 1.6. You should not need Go installed locally to run this script.
# The Go build is done separately from the Docker build to avoid bloating the
# production image with the Go compiler.

echo "BUILD"
docker run --rm -v "$PWD/go":/go -e "GOPATH=/go" -e "GOOS=linux" -e "GOARCH=amd64" golang:1.6 go build -o /go/bin/linux_amd64/package-index package-index
docker run --rm -v "$PWD/go":/go -e "GOPATH=/go" -e "GOOS=linux" -e "GOARCH=amd64" golang:1.6 go build -o /go/bin/linux_amd64/test-suite test-suite
cp go/bin/linux_amd64/package-index docker/package-index/package-index
cp go/bin/linux_amd64/test-suite docker/test-suite/test-suite
docker build -q -t package-index docker/package-index
docker build -q -t test-suite docker/test-suite

echo "UNIT TESTS"
docker run --rm -v "$PWD/go":/go -e "GOPATH=/go" golang:1.6 go test -cover package-index/... test-suite/...

echo "FUNCTIONAL TESTS"
SVR_CID=$(docker run -d package-index /package-index -addr :8080)
SVR_IP=$(docker inspect --format '{{ .NetworkSettings.IPAddress }}' ${SVR_CID})
docker run --rm test-suite /test-suite -addr "${SVR_IP}:8080"
docker run --rm test-suite /test-suite -addr "${SVR_IP}:8080" -seed 3 -concurrency 100 -unluckiness 25
docker run --rm test-suite /test-suite -addr "${SVR_IP}:8080" -seed 4 -concurrency 200 -unluckiness 75
docker rm -f "${SVR_CID}"

echo "BENCHMARKS"
docker run --rm -v "$PWD/go":/go -e "GOPATH=/go" golang:1.6 go test -run=none -bench=. -benchmem package-index/... test-suite/...
