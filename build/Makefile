BIN := zk-controller
OS := linux
OUTPUT_DIR := ../bin

ifeq ($(REGISTRY),)
    REGISTRY := index-dev.qiniu.io/kelibrary
endif

ifeq ($(VERSION),)
    VERSION := latest
endif

TEMP_DIR := $(shell mktemp -d -t $(BIN))

IMAGE = $(REGISTRY)/$(BIN):$(VERSION)
IMAGE_LATEST = $(REGISTRY)/$(BIN):latest

build:
	cp -r ./* $(TEMP_DIR)
	cp $(OUTPUT_DIR)/$(OS)/$(BIN) $(TEMP_DIR)
	chmod a+rx $(TEMP_DIR)/$(BIN)

	docker build -t ${IMAGE} $(TEMP_DIR)
	docker tag ${IMAGE} ${IMAGE_LATEST}
	rm -rf $(TEMP_DIR)
.PHONY: build

push:
	docker push ${IMAGE}
.PHONY: push

push-latest:
	docker push ${IMAGE_LATEST}
.PHONY: push-latest
