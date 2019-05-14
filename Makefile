.DEFAULT_GOAL := build

DEP = github.com/golang/dep/cmd/dep
DEP_CHECK := $(shell command -v dep 2> /dev/null)
GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

ansi_red=\033[0;31m
ansi_grn=\033[0;32m
ansi_yel=\033[0;33m
ansi_end=\033[0m

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: deps
	@echo "$(ansi_grn)Building...$(ansi_end)"
	go build -o enonic github.com/enonic/enonic-cli/internal/app

.PHONY: clean
clean:
	@echo "$(ansi_grn)Cleaning...$(ansi_end)"
	rm -rf dist/*
	rm -rf vendor/*

.PHONY: dep
dep:
ifndef DEP_CHECK
	@echo "$(ansi_grn)Installing dep...$(ansi_end)"
	go get -v $(DEP)
endif

.PHONY: deps
deps: dep
	@echo "$(ansi_grn)Installing vendor dependencies...$(ansi_end)"
	dep ensure -v
