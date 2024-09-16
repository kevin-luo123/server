GO = go

# Define the source files and output binaries
CLIENT_SRC = ./cmd/snowcast_control/main.go
CLIENT_BIN = snowcast_control

SERVER_SRC = ./cmd/snowcast_server/main.go
SERVER_BIN = snowcast_server

# Default target
all: $(CLIENT_BIN) $(SERVER_BIN)

# Rule to build the client control executable
$(CLIENT_BIN): $(CLIENT_SRC)
	$(GO) build -o $@ $(CLIENT_SRC)

# Rule to build the server executable
$(SERVER_BIN): $(SERVER_SRC)
	$(GO) build -o $@ $(SERVER_SRC)

# Clean up build artifacts
clean:
	rm -f $(CLIENT_BIN) $(SERVER_BIN)

.PHONY: all clean