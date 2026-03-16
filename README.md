# gsq

Query game servers from the command line or Go code. Supports 30+ games across Source engine, Minecraft (Java & Bedrock), EOS, and TShock protocols, with auto-detection and host scanning.

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
gsq --rules --game tf2 192.168.1.100:27015   # include server rules/cvars
gsq --json --game minecraft play.hypixel.net # JSON output
gsq games                                    # list supported games, ports, and capabilities
gsq scan 192.168.1.100                       # find servers on known query ports
gsq scan --ports 25000-26000 192.168.1.100   # scan a custom port range
```

> **Tip:** Use `--direct` when you know the exact query port. This skips all port derivation and gives the fastest, most predictable result.

Not all games support every flag — `gsq games` shows what each game supports. Using an incompatible flag with `--game` will error immediately rather than timing out.

### Ports

The **game port** is always the port you provided — the port players connect to. The **query port** is the port that responded to the protocol query. These may differ when gsq derives the query port from a game port (e.g. `--game rust 192.168.1.100:28015` queries on `28017`). gsq does not guess the game port from the query port, as containerized setups make offset-based inference unreliable.

## Library

```go
import "github.com/0xkowalskidev/gsq"

server, err := gsq.Query(ctx, "play.hypixel.net", 25565, gsq.QueryOptions{Game: "minecraft"})
server, err := gsq.Query(ctx, "192.168.1.100", 27015, gsq.QueryOptions{Game: "tf2", Players: true, Rules: true})
server, err := gsq.Query(ctx, "192.168.1.100", 28017, gsq.QueryOptions{Game: "rust", Direct: true})
servers, err := gsq.Discover(ctx, "192.168.1.100", gsq.DiscoverOptions{})
```

## EOS (Epic Online Services)

Some games (ARK: Survival Ascended, Palworld, Squad, The Isle: EVRIMA) use Epic's matchmaking API instead of a direct query protocol. gsq ships with default credentials, but game updates may rotate them.

```bash
gsq --game asa 1.2.3.4:7777                                                  # uses built-in credentials
gsq --game asa --eos-client-id X --eos-client-secret Y 1.2.3.4:7777          # override credentials
GSQ_EOS_CLIENT_ID=X GSQ_EOS_CLIENT_SECRET=Y gsq --game asa 1.2.3.4:7777      # or via env vars
```

If the built-in credentials stop working after a game update, you can extract current credentials from the game's files or find them online.

## License

MIT
