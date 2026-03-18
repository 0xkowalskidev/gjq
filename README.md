# gjq - GameJanitor Query

Query game servers from the command line or Go code. Supports 30+ games across Source engine, Minecraft (Java & Bedrock), EOS, and TShock protocols, with auto-detection and host scanning.

## Install

```bash
go install github.com/0xkowalskidev/gjq/cmd/gjq@latest
```

## Usage

```bash
gjq --game rust 192.168.1.100:28017          # game + query port (fastest)
gjq 192.168.1.100:28017                      # auto-detect protocol
gjq --game rust 192.168.1.100                # infer default query port from game
gjq 192.168.1.100:28015                      # auto-detect with port derivation
gjq --direct 192.168.1.100:28017             # skip port derivation, query exact port
gjq --players --game ark 192.168.1.100:27015 # include player list
gjq --rules --game tf2 192.168.1.100:27015   # include server rules/cvars
gjq --protocol source 192.168.1.100:28017    # force protocol, expects query port
gjq --json --game minecraft play.hypixel.net # JSON output
gjq games                                    # list supported games, ports, and capabilities
gjq scan 192.168.1.100                       # find servers on known query ports
gjq scan --ports 25000-26000 192.168.1.100   # scan a custom port range
```

> **Tip:** Use `--direct` when you know the exact query port. This skips all port derivation and gives the fastest, most predictable result.

Not all games support every flag — `gjq games` shows what each game supports. Using an incompatible flag with `--game` will error immediately rather than timing out.

### Ports

The **game port** is always the port you provided — the port players connect to. The **query port** is the port that responded to the protocol query. These may differ when gjq derives the query port from a game port (e.g. `--game rust 192.168.1.100:28015` queries on `28017`). gjq does not guess the game port from the query port, as containerized setups make offset-based inference unreliable.

## Library

```go
import "github.com/0xkowalskidev/gjq"

server, err := gjq.Query(ctx, "play.hypixel.net", 25565, gjq.QueryOptions{Game: "minecraft"})
server, err := gjq.Query(ctx, "192.168.1.100", 27015, gjq.QueryOptions{Game: "tf2", Players: true, Rules: true})
server, err := gjq.Query(ctx, "192.168.1.100", 28017, gjq.QueryOptions{Game: "rust", Direct: true})
servers, err := gjq.Discover(ctx, "192.168.1.100", gjq.DiscoverOptions{})
```

### Scanning

`gjq scan` is designed for querying your own hosts — e.g. finding what game servers are running on a machine you control. Only scan hosts you own or have permission to probe.

## EOS (Epic Online Services)

Some games (ARK: Survival Ascended, Palworld, Squad, The Isle: EVRIMA) use Epic's matchmaking API instead of a direct query protocol. gjq ships with default credentials, but game updates may rotate them.

```bash
gjq --game asa 1.2.3.4:7777                                                  # uses built-in credentials
gjq --game asa --eos-client-id X --eos-client-secret Y 1.2.3.4:7777          # override credentials
GJQ_EOS_CLIENT_ID=X GJQ_EOS_CLIENT_SECRET=Y gjq --game asa 1.2.3.4:7777      # or via env vars
```

If the built-in credentials stop working after a game update, you can extract current credentials from the game's files or find them online.

## License

MIT
