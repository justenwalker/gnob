gnobsrc := $(wildcard *.go)

all: gnob

gnob: $(gnobsrc)
	@echo "Building gnob"
	@go build -o gnob -tags gnob ./...

.PHONY: test
test: gnob
	./gnob