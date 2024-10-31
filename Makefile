GOOS := linux
GOARCH := amd64
LDFLAGS := "-s -w"
OUTPUT_DIR := bin

.PHONY: build clean

build:
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags $(LDFLAGS) -o $(OUTPUT_DIR)/k8s-bootstrapper .

clean:
	@rm -rf $(OUTPUT_DIR)
