VERSION:= $(shell git describe --tags --dirty --always)
BINS := $(addprefix bin/,$(shell ls cmd))

.PHONY: bins
bins: $(BINS)

.PHONY: $(BINS)
$(BINS): bin
	 go build \
		 -i \
		 -v \
		 -o $@ \
		 -ldflags "-X main.version=$(VERSION)" \
		 ./$(subst bin,cmd,$@)

bin:
	mkdir -p bin

.PHONY: test
test:
	go test $(GO_BUILD_ALL) -v -p 1 ./...
