# Go Relay 

[![Build Status](https://drone-kr.webrelay.io/api/badges/webhookrelay/relay-go/status.svg)](https://drone-kr.webrelay.io/webhookrelay/relay-go)

Lightweight and Open Source Webhook Relay forwarding client.


## Installation

This client requires Go 1.12 (earlier versions might work as well) to be installed ([install instructions](https://golang.org/doc/install)) on your system.

1.  After the installation, find out the GOPATH using `go env`, and then setup a source folder there

```bash
mkdir $GOPATH/src
```

2. Clone relay-go project into your Go source directory:

```bash
cd $GOPATH/src
mkdir -p github.com/webhookrelay/
cd github.com/webhookrelay/
git clone https://github.com/webhookrelay/relay-go
```

3. Finally, to install client application, run:

```bash
cd relay-go
make install
```

## Compiling binaries for Linux and Windows

To compile your binaries for Linux (64-bit) and Windows (64-bit):

```bash
make release
```

Binaries will be available in the `cmd/relayd/release` directory.

## Forwarding webhooks

As opposed to the [Webhook Relay CLI](https://webhookrelay.com/v1/installation/cli), this client will not auto-create buckets, inputs and destinations when `forward` command is used. It will simply connect to Webhook Relay [WebSocket server](https://webhookrelay.com/v1/guide/socket-server) and subscribe to an already created buckets. 

To start forwarding bucket **foo** webhooks, first set several environment variables:

```bash
export RELAY_KEY=your-token-key
export RELAY_SECRET=your-token-secret
export BUCKETS=foo
```

Then:

```bash
relayd forward
```

Alternatively, you can set these variables through command line flags:

```bash
relayd --key your-token-key --secret your-token-secret forward --bucket foo
```

## Test

To run all tests:

```
make test
```
