#!/bin/bash

# Build the TCP Chat server
echo "Building TCP Chat server..."
go build -o TCPChat ./cmd

if [ $? -eq 0 ]; then
    echo "Build successful! Binary created as 'TCPChat'"
    echo "Run './TCPChat' to start the server with default port 8989"
    echo "Or './TCPChat [port]' to specify a custom port"
else
    echo "Build failed!"
    exit 1
fi
