# Define the output binaries
SERVER_BINARY=server
CLIENT_BINARY=client

# Define the source directories
SERVER_SRC=./cmd/server/main.go
CLIENT_SRC=./cmd/client/main.go

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

.PHONY: all build build-server build-client clean
