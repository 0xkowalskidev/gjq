# gjq - GameJanitor Query

Query game servers from the command line or Go code. Supports 75+ games across Source engine, Minecraft (Java & Bedrock), EOS, and TShock protocols, with auto-detection and host scanning.

## Install

**Prebuilt binary:**
```bash
curl -sSL https://raw.githubusercontent.com/0xkowalskidev/gjq/master/install.sh | sh
```

**Go:**
```bash
go install github.com/0xkowalskidev/gjq/cmd/gjq@latest
```

**Nix:**
```bash
nix run github:0xkowalskidev/gjq         # run without installing
nix profile install github:0xkowalskidev/gjq  # install to profile
```

## Usage

```bash
gjq --game rust 192.168.1.100:28015          # query by game (derives query port)
gjq --game rust 192.168.1.100                # omit port to use game's default
gjq 192.168.1.100:27015                      # auto-detect protocol and game
gjq --direct 192.168.1.100:28017             # exact query port, auto-detect protocol
gjq --protocol source 192.168.1.100:28017    # exact query port, force protocol
gjq --players --game ark 192.168.1.100:27015 # include player list
gjq --rules --game tf2 192.168.1.100:27015   # include server rules/cvars
gjq --json --game minecraft play.hypixel.net # JSON output
gjq games --json                             # list supported games as JSON
gjq scan 192.168.1.100                       # find servers on known query ports
gjq scan --ports 25000-26000 192.168.1.100   # scan a custom port range
```

`--protocol` is useful for querying games not yet in the registry — any A2S game works with `--protocol source`.

Not all games support every flag — `gjq games` shows what each game supports. Using an incompatible flag with `--game` will error immediately rather than timing out.

### Ports

The **query port** is always accurate — it's the port that actually responded. The **game port** may not be — most protocols report it, but containerized servers (Docker, k8s) often expose different ports than the server thinks it's running on. gjq shows the port you provided rather than the server-reported value, but even that is only as correct as your input and any port derivation gjq applied.

## Library

```go
import "github.com/0xkowalskidev/gjq"

server, err := gjq.Query(ctx, "play.hypixel.net", 25565, gjq.QueryOptions{Game: "minecraft"})
server, err := gjq.Query(ctx, "192.168.1.100", 27015, gjq.QueryOptions{Game: "tf2", Players: true, Rules: true})
server, err := gjq.Query(ctx, "192.168.1.100", 28017, gjq.QueryOptions{Game: "rust", Direct: true})
server, err := gjq.Query(ctx, "192.168.1.100", 28017, gjq.QueryOptions{Protocol: "source"})
servers, err := gjq.Discover(ctx, "192.168.1.100", gjq.DiscoverOptions{})
```

### Scanning

`gjq scan` / `gjq.Discover` probes ports to find game servers on a host. Primarily intended for localhost — results over the network may be unreliable with large port ranges.

## Response Format

### ServerInfo

| Field | Type | JSON | Notes |
|-------|------|------|-------|
| Protocol | string | `protocol` | Protocol used: `source`, `minecraft`, `raknet`, `eos`, `tshock`, `fivem` |
| Name | string | `name` | Server name or MOTD |
| Map | string | `map` | Current map |
| Game | string | `game` | Game name, matched from the game registry — not server-reported |
| GameMode | string | `gameMode` | Game mode (Bedrock only) |
| Players | int | `players` | Current player count |
| MaxPlayers | int | `maxPlayers` | Maximum player slots |
| Bots | int | `bots` | Bot count |
| ServerType | string | `serverType` | `dedicated`, `non-dedicated`, or `proxy` (Source only) |
| Environment | string | `environment` | OS: `linux`, `windows`, `mac` (Source only) |
| Visibility | string | `visibility` | `public` or `private` |
| VAC | bool | `vac` | VAC enabled (Source only) |
| Version | string | `version` | Server/game version string |
| Keywords | string | `keywords` | Comma-separated tags set by the server (Source only) |
| AppID | uint32 | `appId` | Steam AppID (Source only) |
| Ping | duration | `ping` | Round-trip query time |
| Address | string | `address` | Address as provided by the user |
| GamePort | uint16 | `gamePort` | User-provided port — may not be the actual game port |
| ReportedGamePort | uint16 | `reportedGamePort` | Game port server thinks it's running on, unreliable |
| QueryPort | uint16 | `queryPort` | Port that responded to the query — always accurate |
| PlayerList | []PlayerInfo | `playerList` | Requires `--players`. Not all games support this |
| Rules | map[string]string | `rules` | Server cvars/rules. Requires `--rules` (Source only) |
| Mods | []ModInfo | `mods` | Mod list (Minecraft Java only) |
| Extra | map[string]any | `extra` | Protocol-specific data (steamId, edition, folder, etc.) |

### PlayerInfo

| Field | Type | JSON | Notes |
|-------|------|------|-------|
| Name | string | `name` | Player name. EOS returns opaque IDs instead of names |
| Score | int | `score` | Player score (Source only, 0 for other protocols) |
| Duration | duration | `duration` | Time connected (Source only) |

### ModInfo

| Field | Type | JSON | Notes |
|-------|------|------|-------|
| ID | string | `id` | Mod identifier |
| Version | string | `version` | Mod version |

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
