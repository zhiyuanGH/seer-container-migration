# Define the output binaries
SERVER_BINARY=ectrserver
CLIENT_BINARY=ectr

# Define the source directories
SERVER_SRC=./cmd/ectrserver/main.go
CLIENT_SRC=./cmd/ectr/main.go

# Define the installation directory
INSTALL_DIR=/usr/local/bin

# Default target
all: build

# Build both server and client
build: build-server build-client

# Build the server binary
build-server:
	go build -o $(SERVER_BINARY) $(SERVER_SRC)

# Build the client binary
build-client:
	go build -o $(CLIENT_BINARY) $(CLIENT_SRC)

# Clean the build artifacts
clean:
	rm -f $(SERVER_BINARY) $(CLIENT_BINARY)

# Install the binaries to the system's bin directory
install: build
	cp $(SERVER_BINARY) $(INSTALL_DIR)
	cp $(CLIENT_BINARY) $(INSTALL_DIR)

.PHONY: all build build-server build-client clean install

