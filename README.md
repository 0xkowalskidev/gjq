# gsq

Query game servers from the command line or Go code. Supports Source engine (CS2, Rust, Ark, etc.) and Minecraft, with auto-detection and host scanning.

## Install

```bash
go install github.com/0xkowalskidev/gsq/cmd/gsq@latest
```

## Usage

```bash
gsq --game rust 192.168.1.100:28017          # game + query port (fastest)
gsq 192.168.1.100:28017                      # auto-detect protocol
gsq --game rust 192.168.1.100                # infer default query port from game
gsq 192.168.1.100:28015                      # auto-detect with port derivation
gsq --direct 192.168.1.100:28017             # skip port derivation, query exact port
gsq --players --game ark 192.168.1.100:27015 # include player list
gsq --json --game minecraft play.hypixel.net # JSON output
gsq games                                    # list supported games
gsq scan 192.168.1.100                       # find servers on known query ports
gsq scan --ports 25000-26000 192.168.1.100   # scan a custom port range
```

> **Tip:** Use `--direct` when you know the exact query port. This skips all port derivation and gives the fastest, most predictable result.

Output includes the address you queried, the inferred game port (what players connect to), and the query port (what the protocol responded on):

```
Name:        Rust Server
Address:     192.168.1.100
Game Port:   28015
Query Port:  28017
Game:        Rust
...
```

> **Note:** The **game port** is always the port you provided — this is the port players would use to connect. The **query port** is the port that actually responded to the protocol query. These may differ when gsq derives the query port from a game port you provided (e.g. you give `--game rust 192.168.1.100:28015` and gsq finds the query protocol responding on `28017`). gsq does not attempt to guess the game port from the query port, as containerized and non-standard server setups make offset-based inference unreliable.

## Library

```go
import "github.com/0xkowalskidev/gsq"

server, err := gsq.Query(ctx, "play.hypixel.net", 25565, gsq.QueryOptions{Game: "minecraft"})
server, err := gsq.Query(ctx, "192.168.1.100", 27015, gsq.QueryOptions{Game: "ark", Players: true})
server, err := gsq.Query(ctx, "192.168.1.100", 28017, gsq.QueryOptions{Game: "rust", Direct: true})
servers, err := gsq.Discover(ctx, "192.168.1.100", gsq.DiscoverOptions{})
```

Set `Direct: true` when you know the exact query port to skip port derivation.

## EOS (Epic Online Services)

Some games (ARK: Survival Ascended, Palworld, Squad, The Isle: EVRIMA) use Epic's matchmaking API
instead of a direct query protocol. gsq ships with default credentials for these games,
but game updates may rotate them.

```bash
gsq --game asa 1.2.3.4:7777                    # uses built-in credentials
gsq --game asa --eos-client-id X --eos-client-secret Y 1.2.3.4:7777  # override credentials
```

If the built-in credentials stop working after a game update, you can extract current
credentials from the game's files or find them on community resources like opengsq.

> **Limitation:** The EOS matchmaking API returns at most 15 sessions per request with no
> pagination. If the server's IP hosts many game instances (common with large hosting
> providers), the target server may not appear in the results and the query will fail.

## Terraria (TShock)

Requires [TShock](https://github.com/Pryaxis/TShock) — vanilla Terraria has no query protocol.
The server must have `RestApiEnabled: true` in its TShock config and query port must be open.

## License

MIT
