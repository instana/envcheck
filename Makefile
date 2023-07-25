SHELL := /bin/bash -eu -o pipefail
CMDS := $(wildcard cmd/*)
IMGS := ${DOCKER_REPO}/envcheck-pinger ${DOCKER_REPO}/envcheck-daemon
SRC := $(shell find . -name \*.go) # not win compatible but :shrug:
GIT_SHA := $(shell git rev-parse --short HEAD)
GO_LINUX := GOOS=linux GOARCH=amd64 go
GO_OSX := GOOS=darwin GOARCH=amd64 go
GO_WIN64 := GOOS=windows GOARCH=amd64 go
EXE := envcheckctl.amd64 envcheckctl.exe envcheckctl.darwin64 envcheck-pinger envcheck-daemon envcheck-repocheck

.PHONY: all
all: vet lint coverage envcheckctl

.PHONY: publish
publish: push-daemon push-pinger push-repocheck

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
	go install golang.org/x/lint/golint@latest

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

envcheck-repocheck: $(SRC)
	$(GO_LINUX) build -v -ldflags "-X main.Revision=$(GIT_SHA)" -o $@ ./cmd/repocheck

.PHONY: docker
docker: docker-daemon docker-pinger docker-repocheck

.PHONY: docker-promep
docker-promep:
	docker build . -f cmd/promep/Dockerfile -t  $(DOCKER_REPO)/promep:latest -t $(DOCKER_REPO)/promep:${GIT_SHA}

.PHONY: push-promep
push-promep: docker-promep
	docker push $(DOCKER_REPO)/promep:${GIT_SHA}
	docker push $(DOCKER_REPO)/promep:latest

.PHONY: docker-daemon
docker-daemon: envcheck-daemon
	docker build . -t $(DOCKER_REPO)/envcheck-daemon:latest -t $(DOCKER_REPO)/envcheck-daemon:${GIT_SHA} --build-arg CMD_PATH=./envcheck-daemon

.PHONY: push-daemon
push-daemon: docker-daemon
	docker push $(DOCKER_REPO)/envcheck-daemon:${GIT_SHA}
	docker push $(DOCKER_REPO)/envcheck-daemon:latest

.PHONY: docker-pinger
docker-pinger: envcheck-pinger
	docker build . -t $(DOCKER_REPO)/envcheck-pinger:latest -t $(DOCKER_REPO)/envcheck-pinger:${GIT_SHA} --build-arg CMD_PATH=./envcheck-pinger

.PHONY: push-pinger
push-pinger: docker-pinger
	docker push $(DOCKER_REPO)/envcheck-pinger:${GIT_SHA}
	docker push $(DOCKER_REPO)/envcheck-pinger:latest

.PHONY: docker-repocheck
docker-repocheck: envcheck-repocheck
	docker build . -t $(DOCKER_REPO)/envcheck-repocheck:latest -t $(DOCKER_REPO)/envcheck-repocheck:${GIT_SHA} --build-arg CMD_PATH=./envcheck-repocheck

.PHONY: push-repocheck
push-repocheck: docker-repocheck
	docker push $(DOCKER_REPO)/envcheck-repocheck:${GIT_SHA}
	docker push $(DOCKER_REPO)/envcheck-repocheck:latest

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
	rm -f *.out $(EXE)
	go clean -i
