# Define the output binaries
SERVER_BINARY=ectrserver
CLIENT_BINARY=ectr
EXP_COLLECT_PF_BINARY=ectrpf
EXP_COLLECT_MIGRATION_TIME_BINARY=ectrt

# Define the source directories
SERVER_SRC=./cmd/ectrserver/main.go
CLIENT_SRC=./cmd/ectr/main.go
EXP_COLLECT_PF_SRC =./cmd/expCollectPF/main.go
EXP_COLLECT_MIGRATION_TIME_SRC =./cmd/expCollectT/main.go

# Define the installation directory
INSTALL_DIR=/usr/local/bin

# Default target
all: build

# Build both server and client
build: build-server build-client build-exp-collect-pf build-exp-collect-t

# Build the exp_collect_t binary
build-exp-collect-t:
	go build -o $(EXP_COLLECT_MIGRATION_TIME_BINARY) $(EXP_COLLECT_MIGRATION_TIME_SRC)

# Build the exp_collect_pf binary
build-exp-collect-pf:
	go build -o $(EXP_COLLECT_PF_BINARY) $(EXP_COLLECT_PF_SRC)

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
	cp $(EXP_COLLECT_PF_BINARY) $(INSTALL_DIR)
	cp $(EXP_COLLECT_MIGRATION_TIME_BINARY) $(INSTALL_DIR)

.PHONY: all build build-server build-client clean install

