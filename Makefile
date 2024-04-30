.PHONY: all build clean local

# Define platforms for cross-compilation
PLATFORMS := windows/amd64 \
			 windows/arm64 \
             linux/amd64 \
             darwin/amd64 \
             linux/arm64 \
             darwin/arm64
BUILD_DIR := build

# Default target builds for local architecture
all: clean local

# Build binary for local architecture
local:
	@echo "Building for local architecture..."
	@go build -o $(BUILD_DIR)/llm-router-local cmd/main.go

# Build binaries for all platforms
build:
	@mkdir -p $(BUILD_DIR)
	@$(foreach PLATFORM,$(PLATFORMS),\
		$(eval GOOS=$(word 1,$(subst /, ,$(PLATFORM))))\
		$(eval GOARCH=$(word 2,$(subst /, ,$(PLATFORM))))\
		$(eval OUTPUT=$(BUILD_DIR)/llm-router-$(GOOS)-$(GOARCH))\
		$(if $(findstring windows,$(GOOS)), $(eval OUTPUT:=$(OUTPUT).exe))\
		echo "Building for $(GOOS)/$(GOARCH)..." && \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(OUTPUT) cmd/main.go;)

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)