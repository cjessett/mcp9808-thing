# mcp9808-thing

This script reads an MCP9808 temperature sensor from a Raspberry Pi and updates an AWS Iot Thing Shadow.

## Prerequisites

Create a `.env` file with the following:
```
THING_NAME=<Thing name>
ENDPOINT=<Thing endpoint>
PRIVATE_KEY_PATH=<path to private key>
CERT_PATH=<path to cert>
ROOT_CA_PATH=<path to root ca>
LOG_FILE_PATH=<path to log file>
```

## Building

For a Raspberry Pi Zero
```
env GOOS=linux GOARCH=arm GOARM=6 go build main.go
```

## Execute the script
```
./main
```