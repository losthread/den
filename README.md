# prox

Prox is a fast, private peer-to-peer messenger for direct communication between devices, written in Go.

## Goals

- Communicate directly between peers without any central server
- Keep conversations private and local to participating devices
- No centralised database involved
- Provide a fast, lightweight client with a simple user experience

## Status

Early development. The project is currently being built; setup and usage instructions will be added as the first working release takes shape.

## Project layout

```text
cmd/peer/          Peer process and TCP connection entry point
cmd/tui/           Terminal user interface
internal/peer/     Shared peer networking code
web/               Future web client
```

## License

MIT
