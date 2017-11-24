VERSION:= $(shell git describe --tags --dirty --always)
BINS := $(addprefix bin/,$(shell ls cmd))

.PHONY: default
default: bin/netfix

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

.PHONY: build
build: bin/netfix
	mkdir -p build/bin
	cp bin/netfix build/bin
	cp config.sh pi_install.sh netfix.service build
	cp -R www migrations build

bin:
	mkdir -p bin

.PHONY: test
test:
	go test $(GO_BUILD_ALL) -v -p 1 ./...

.PHONY: clean
clean:
	rm -rf bin build
