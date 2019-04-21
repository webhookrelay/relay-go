# Go Relay

Lightweight and Open Source Webhook Relay forwarding client.


## Installation

This client requires Go to be installed on your system. To install client application, run:

```
make install
```

## Forwarding webhooks

As opposed to the [Webhook Relay CLI](https://webhookrelay.com/v1/installation/cli), this client will not auto-create buckets, inputs and destinations when `forward` command is used. It will simply connect to Webhook Relay [WebSocket server](https://webhookrelay.com/v1/guide/socket-server) and subscribe to an already created buckets. 

To start forwarding bucket **foo** webhooks, first set several environment variables:

```bash
export RELAY_KEY=your-token-key
export RELAY_SECRET=your-token-secret
export BUCKETS=foo
```

Then:

```
relayd forward --buckets foo  
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

## TODO

- [ ] authentication
- [ ] webhook forwarding
- [ ] build pipeline
- [ ] e2e tests
- [ ] webhook binary payload tests