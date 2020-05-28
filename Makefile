SHELL := /bin/bash -eu
CMDS := $(wildcard cmd/*)
IMGS := ${DOCKER_REPO}/envcheck-pinger ${DOCKER_REPO}/envcheck-daemon
SRC := $(wildcard cmd/**/*.go) $(wildcard *.go)
GIT_SHA := $(shell git rev-parse --short HEAD)

.PHONY: all
all: vet lint coverage envcheckctl

.PHONY: publish
publish: all $(IMGS)

.PHONY: test
test: cover.out

.PHONY: vet
vet: vet.out

.PHONY: lint
lint: lint.out

.PHONY: coverage
coverage: coverage.out

.PHONY: envcheckctl
envcheckctl: envcheckctl.amd64 envcheckctl.exe envcheckctl.darwin64

envcheckctl.exe: $(SRC)
	GOOS=windows GOARCH=amd64 go build -v -ldflags "-X main.Revision=$(GIT_SHA)" -o $@ ./cmd/envcheckctl

envcheckctl.darwin64: $(SRC)
	GOOS=darwin GOARCH=amd64 go build -v -ldflags "-X main.Revision=$(GIT_SHA)" -o $@ ./cmd/envcheckctl

envcheckctl.amd64: $(SRC)
	GOOS=linux GOARCH=amd64 go build -v -ldflags "-X main.Revision=$(GIT_SHA)" -o $@ ./cmd/envcheckctl

# run the tests with atomic coverage
cover.out: $(SRC)
	go test -v -cover -covermode atomic -coverprofile cover.out ./...

# generate the HTML coverage report
coverage.html: cover.out
	go tool cover -html=cover.out -o coverage.html

# generate the text coverage summary
coverage.out: cover.out
	go tool cover -func=cover.out | tee coverage.out

# run vet against the codebase
vet.out: $(SRC)
	go vet github.com/instana/envcheck/... | tee vet.out

# run the linter against the codebase
lint.out: $(SRC)
	golint ./... | tee lint.out

# clean the generated files
.PHONY: clean
clean:
	rm -f *.out envcheckctl.*
	go clean -i ./...

# build a docker container per service
.PHONY: ${DOCKER_REPO}/
${DOCKER_REPO}/%:
	docker build . -t $@:latest -t $@:${GIT_SHA} --build-arg CMD_PATH=./cmd/$(subst ${DOCKER_REPO}/envcheck-,,$@) --build-arg GIT_SHA=${GIT_SHA}
	docker push $@:${GIT_SHA}
	docker push $@:latest
