gnobsrc := $(wildcard *.go)

all: gnob

gnob: $(gnobsrc)
	@echo "Building gnob"
	@go build -o gnob -tags gnob ./...

.PHONY: test
test: gnob
	@echo "--- Listing Targets"
	./gnob -help
	@echo "--- Help on Target 'default'"
	./gnob -help default
	@echo "--- Run 'default'"
	./gnob default