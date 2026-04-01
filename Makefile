BINARY_NAME ?= appstore
VERSION ?= dev
DIST_DIR ?= dist

.PHONY: build run tidy brew-dist clean

build:
	go build -o bin/$(BINARY_NAME) .

run:
	go run .

tidy:
	go mod tidy

brew-dist: clean
	mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME) .
	tar -czf $(DIST_DIR)/$(BINARY_NAME)_$(VERSION)_darwin_amd64.tar.gz $(BINARY_NAME)
	rm $(BINARY_NAME)
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME) .
	tar -czf $(DIST_DIR)/$(BINARY_NAME)_$(VERSION)_darwin_arm64.tar.gz $(BINARY_NAME)
	rm $(BINARY_NAME)
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) .
	tar -czf $(DIST_DIR)/$(BINARY_NAME)_$(VERSION)_linux_amd64.tar.gz $(BINARY_NAME)
	rm $(BINARY_NAME)
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_NAME) .
	tar -czf $(DIST_DIR)/$(BINARY_NAME)_$(VERSION)_linux_arm64.tar.gz $(BINARY_NAME)
	rm $(BINARY_NAME)
	cd $(DIST_DIR) && shasum -a 256 *.tar.gz > checksums.txt
	@echo "brew-dist artifacts ready in $(DIST_DIR)"

clean:
	rm -rf $(DIST_DIR) bin
