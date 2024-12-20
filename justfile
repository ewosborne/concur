export bin := "concur"

set dotenv-load := false

# test for quoted args to work, didn't do anything. doesn't matter much.

#set positional-arguments := true

default:
    just --list

coverage:
    go test ./cmd -coverprofile=coverage.out
    go tool cover -html=coverage.out

build:
    goreleaser build --single-target --snapshot --clean
    #ln -fs dist/{{ bin }}_darwin_arm64_v8.0/{{ bin }} ./$bin
    cp dist/{{ bin }}_darwin_arm64_v8.0/{{ bin }} .

test:
    go test ./cmd

testv:
    go test ./cmd -test.v

fmt:
    just --unstable --fmt
    goimports -l -w .
    go fmt

mac: test build


clean:
    go clean -testcache
    go mod tidy
    rm -f $bin 
    rm -rf dist

install: mac
    cp ./$bin ~/bin/

release arg1: testv
   rm -rf dist/
   git tag {{ arg1 }}
   git push origin {{ arg1 }}
   goreleaser release
