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

test: build
    go test ./tests

testv: build
    go test ./tests -test.v

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

require-env:
    #!/usr/bin/env sh
    if [ -z "${GITHUB_TOKEN:-}" ]; then
        echo "Error: GITHUB_TOKEN environment variable is not set"
        exit 1
    fi

release arg1: require-env testv
    rm -rf dist/
    echo "{{ arg1 }}" > ./.version
    git tag {{ arg1 }}
    git push origin {{ arg1 }}
    goreleaser release
