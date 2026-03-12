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
gsq --players --game ark 192.168.1.100:27015   # include player list
gsq --json --game minecraft play.hypixel.net   # JSON output
gsq scan 192.168.1.100                         # find all game servers on a host
gsq scan --ports 25000-26000 192.168.1.100     # scan custom port range
gsq games                                      # list supported games
```

Output includes the address you queried, the inferred game port (what players connect to), and the query port (what the protocol responded on):

```
Name:        Rust Server
Address:     192.168.1.100
Game Port:   28015
Query Port:  28017
Game:        Rust
...
```

> **Note:** The game port is inferred from the query port using the known offset for each game (e.g. Rust query port = game port + 2). For servers with non-standard port layouts, such as containerized servers where port mappings don't preserve the offset, the displayed game port may be incorrect. The query port is always accurate.

## Library

```go
import "github.com/0xkowalskidev/gsq"

server, err := gsq.Query(ctx, "play.hypixel.net", 25565, gsq.QueryOptions{Game: "minecraft"})
server, err := gsq.Query(ctx, "192.168.1.100", 27015, gsq.QueryOptions{Game: "ark", Players: true})
servers, err := gsq.Discover(ctx, "192.168.1.100", gsq.DiscoverOptions{})
```

## Unsupported: EOS (Epic Online Services)

Some games use Epic Online Services for server discovery instead of a direct query protocol. These games are **not supported** because EOS does not allow querying game servers directly — instead, servers push their state to Epic's central API, and clients must authenticate with per-game OAuth2 credentials to read it. This makes queries dependent on Epic's infrastructure and subject to credential rotation whenever a game updates.

Games that use EOS include ARK: Survival Ascended, Squad, Palworld and The Isle.

## License

MIT
