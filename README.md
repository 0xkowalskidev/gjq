# gsq

Query game servers from the command line or Go code. Supports Source engine (CS2, Rust, Ark, etc.) and Minecraft, with auto-detection and host scanning.

## Install

```bash
go install github.com/0xkowalskidev/gsq/cmd/gsq@latest
```

## Usage

```bash
gsq 192.168.1.100:27015                       # auto-detect protocol
gsq --game rust 192.168.1.100                  # specify game, use default port
gsq --json --game minecraft play.hypixel.net   # JSON output
gsq scan 192.168.1.100                         # find all game servers on a host
gsq scan --ports 25000-26000 192.168.1.100     # scan custom port range
gsq games                                      # list supported games
```

## Library

```go
import "github.com/0xkowalskidev/gsq"

server, err := gsq.Query(ctx, "play.hypixel.net", 25565, gsq.QueryOptions{Game: "minecraft"})
servers, err := gsq.Discover(ctx, "192.168.1.100", gsq.DiscoverOptions{})
```

## License

MIT
