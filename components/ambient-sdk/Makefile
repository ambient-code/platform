SPEC_PATH ?= ../ambient-api-server/openapi/openapi.yaml
GO_OUT     = go-sdk
PYTHON_OUT = python-sdk/ambient_platform
GENERATOR  = generator

.PHONY: generate-sdk verify-sdk build-generator clean

build-generator:
	cd $(GENERATOR) && go build -o ../bin/ambient-sdk-generator .

generate-sdk: build-generator
	./bin/ambient-sdk-generator \
		-spec $(SPEC_PATH) \
		-go-out $(GO_OUT) \
		-python-out $(PYTHON_OUT)
	cd $(GO_OUT) && go fmt ./...

verify-sdk: generate-sdk
	cd $(GO_OUT) && go build ./...
	cd python-sdk && python3 -c "from ambient_platform import *"
	@echo "SDK verification passed"

clean:
	rm -f bin/ambient-sdk-generator
