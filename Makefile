BIN            := scoop
OUTPUT_DIR     := build
TMP_DIR        := .tmp
RELEASE_VER    := $(shell git rev-parse --short HEAD)
SEMVER         := $(if $(SEMVER),$(SEMVER),devel)


.PHONY: help
.DEFAULT_GOAL := help

build/linux: semvercheck clean/linux ## Build for linux (save to OUTPUT_DIR/BIN)
	GOOS=linux CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-X main.relver=$(RELEASE_VER) -X main.semver=$(SEMVER)" -o $(OUTPUT_DIR)/$(BIN)-linux .

build/darwin: semvercheck clean/darwin ## Build for darwin (save to OUTPUT_DIR/BIN)
	GOOS=darwin CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-X main.relver=$(RELEASE_VER) -X main.semver=$(SEMVER)" -o $(OUTPUT_DIR)/$(BIN)-darwin .

build/release: semvercheck test/integration build/linux build/darwin ## Prepare a build
	mkdir -p $(OUTPUT_DIR)/${BIN}-$(SEMVER)-darwin
	mkdir -p $(OUTPUT_DIR)/${BIN}-$(SEMVER)-linux
	cp $(OUTPUT_DIR)/$(BIN)-darwin $(OUTPUT_DIR)/${BIN}-$(SEMVER)-darwin/$(BIN)
	cp $(OUTPUT_DIR)/$(BIN)-linux $(OUTPUT_DIR)/${BIN}-$(SEMVER)-linux/$(BIN)
	cd $(OUTPUT_DIR) && tar -czvf ${BIN}-$(SEMVER)-darwin.tgz ${BIN}-$(SEMVER)-darwin/
	cd $(OUTPUT_DIR) && tar -czvf ${BIN}-$(SEMVER)-linux.tgz ${BIN}-$(SEMVER)-linux/
	@echo "A new release has been created!"

semvercheck:
ifeq ($(SEMVER),)
	$(error 'SEMVER' must be set)
endif

clean: clean/darwin clean/linux ## Remove all build artifacts

clean/darwin: ## Remove darwin build artifacts
	$(RM) $(OUTPUT_DIR)/$(BIN)-darwin

clean/linux: ## Remove linux build artifacts
	$(RM) $(OUTPUT_DIR)/$(BIN)-linux

test/integration: ## Intergration Testing
	go test -tags integration -timeout 30s -count=1 -v
	go test ./... -timeout 30s -count=1 -v

docker/build: ## Build Docker Container
	docker build --build-arg SEMVER --build-arg RELEASE_VER -t $(BIN):$(SEMVER) .

docker/run: ## Run Docker Container
	docker run -p 8080:8080/tcp $(BIN):$(SEMVER)

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_\/-]+:.*?## / {printf "\033[34m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
		sort | \
		grep -v '#'	
