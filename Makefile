SHELL := /bin/bash -eu -o pipefail
CMDS := $(wildcard cmd/*)
IMGS := ${DOCKER_REPO}/envcheck-pinger ${DOCKER_REPO}/envcheck-daemon
SRC := $(shell find . -name \*.go) # not win compatible but :shrug:
GIT_SHA := $(shell git rev-parse --short HEAD)
GO_LINUX := GOOS=linux GOARCH=amd64 go
GO_OSX := GOOS=darwin GOARCH=amd64 go
GO_WIN64 := GOOS=windows GOARCH=amd64 go
EXE := envcheckctl.amd64 envcheckctl.exe envcheckctl.darwin64 envcheck-pinger envcheck-daemon

.PHONY: all
all: vet lint coverage envcheckctl

.PHONY: publish
publish: docker-envcheck-daemon docker-envcheck-pinger

.PHONY: test
test: cover.out

.PHONY: vet
vet: vet.out

.PHONY: lint
lint: lint.out

.PHONY: coverage
coverage: coverage.out

.PHONY: envcheckctl
envcheckctl: $(EXE)

.PHONY: deps
deps:
	go get -u golang.org/x/lint/golint

envcheckctl.exe: $(SRC)
	$(GO_WIN64) build -v -ldflags "-X main.Revision=$(GIT_SHA)" -o $@ ./cmd/envcheckctl

envcheckctl.darwin64: $(SRC)
	$(GO_OSX) build -v -ldflags "-X main.Revision=$(GIT_SHA)" -o $@ ./cmd/envcheckctl

envcheckctl.amd64: $(SRC)
	$(GO_LINUX) build -v -ldflags "-X main.Revision=$(GIT_SHA)" -o $@ ./cmd/envcheckctl

envcheck-pinger: $(SRC)
	$(GO_LINUX) build -v -ldflags "-X main.Revision=$(GIT_SHA)" -o $@ ./cmd/pinger

envcheck-daemon: $(SRC)
	$(GO_LINUX) build -v -ldflags "-X main.Revision=$(GIT_SHA)" -o $@ ./cmd/daemon

.PHONY: docker
docker: docker-envcheck-daemon docker-envcheck-pinger

.PHONY: docker-envcheck-daemon
docker-envcheck-daemon: envcheck-daemon
	docker build . -t $(DOCKER_REPO)/envcheck-daemon:latest -t $(DOCKER_REPO)/envcheck-daemon:${GIT_SHA} --build-arg CMD_PATH=./envcheck-daemon

.PHONY: push-envcheck-daemon
push-envcheck-daemon: docker-envcheck-daemon
	docker push $(DOCKER_REPO)/envcheck-daemon:${GIT_SHA}
	docker push $(DOCKER_REPO)/envcheck-daemon:latest

.PHONY: docker-envcheck-pinger
docker-envcheck-pinger: envcheck-pinger
	docker build . -t $(DOCKER_REPO)/envcheck-pinger:latest -t $(DOCKER_REPO)/envcheck-pinger:${GIT_SHA} --build-arg CMD_PATH=./envcheck-pinger

.PHONY: push-envcheck-pinger
push-envcheck-pinger: docker-envcheck-pinger
	docker push $(DOCKER_REPO)/envcheck-pinger:${GIT_SHA}
	docker push $(DOCKER_REPO)/envcheck-pinger:latest

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
	$(shell go list -f {{.Target}} golang.org/x/lint/golint) ./... | tee lint.out

# clean the generated files
.PHONY: clean
clean:
	rm -f *.out $(EXE)
	go clean -i ./...
	