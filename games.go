package gjq

import "github.com/0xkowalskidev/gjq/protocol"

type GameConfig struct {
	Slug             string   `json:"slug"`
	Name             string   `json:"name"`
	Aliases          []string `json:"aliases,omitempty"`
	AppID            uint32   `json:"appId,omitempty"`
	DefaultGamePort  uint16   `json:"defaultGamePort"`
	DefaultQueryPort uint16   `json:"defaultQueryPort"`
	Protocol         string   `json:"protocol"`
	ProtocolConfig   any      `json:"-"`                  // protocol-specific config (e.g. *protocol.EOSConfig), nil for most games
	Supports         []string `json:"supports,omitempty"` // optional features: "players", "rules", "keywords", "mods"
	Notes            string   `json:"notes,omitempty"`    // limitations/quirks shown in `gjq games` and JSON output
}

func (g *GameConfig) HasSupport(feature string) bool {
	for _, s := range g.Supports {
		if s == feature {
			return true
		}
	}
	return false
}

// Shorthand for common support sets to reduce noise in the registry.
var (
	sourceSupports = []string{"players", "rules", "keywords"}
	sourceNoRules  = []string{"players", "keywords"}
	eosSupports    = []string{"players"}
)

// EOS ClientID/ClientSecret are public credentials shipped in each game's binary — they grant
// read-only access to Epic's matchmaking API. They may rotate when a game updates, so users
// can override them with --eos-client-id/--eos-client-secret.
var gameRegistry = []GameConfig{
	{Slug: "minecraft-java", Name: "Minecraft: Java Edition", Aliases: []string{"minecraft", "mc"}, DefaultGamePort: 25565, DefaultQueryPort: 25565, Protocol: "minecraft",
		Supports: []string{"players", "mods"}, Notes: "Player list limited to server's sample (usually 12 max)"},
	{Slug: "minecraft-bedrock", Name: "Minecraft: Bedrock Edition", Aliases: []string{"mcbe", "bedrock"}, DefaultGamePort: 19132, DefaultQueryPort: 19132, Protocol: "raknet"},
	{Slug: "counter-strike-2", Name: "Counter-Strike 2", Aliases: []string{"cs2"}, AppID: 730, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceNoRules},
	{Slug: "counter-strike-go", Name: "Counter-Strike: GO", Aliases: []string{"csgo"}, AppID: 4465480, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports, Notes: "Standalone re-release (March 2026) with new AppID, separate from CS2 (730)"},
	{Slug: "counter-strike-source", Name: "Counter-Strike: Source", Aliases: []string{"css"}, AppID: 240, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "counter-strike", Name: "Counter-Strike 1.6", Aliases: []string{"cs16", "cs"}, AppID: 10, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "rust", Name: "Rust", Aliases: []string{}, AppID: 252490, DefaultGamePort: 28015, DefaultQueryPort: 28017, Protocol: "source", Supports: sourceSupports},
	{Slug: "fivem", Name: "FiveM (GTA V)", Aliases: []string{"gta5"}, DefaultGamePort: 30120, DefaultQueryPort: 30120, Protocol: "fivem", Supports: []string{"players"}, Notes: "Player list may be disabled by server config"},
	{Slug: "cod4", Name: "Call of Duty 4: Modern Warfare", Aliases: []string{"cod4x"}, DefaultGamePort: 28960, DefaultQueryPort: 28960, Protocol: "quake3", Supports: []string{"players"}},
	{Slug: "cod2", Name: "Call of Duty 2", Aliases: []string{}, DefaultGamePort: 28960, DefaultQueryPort: 28960, Protocol: "quake3", Supports: []string{"players"}},
	{Slug: "cod-mw2", Name: "Call of Duty: MW2 (IW4x)", Aliases: []string{"iw4x", "mw2"}, DefaultGamePort: 28960, DefaultQueryPort: 28960, Protocol: "quake3", Supports: []string{"players"}},
	{Slug: "cod-waw", Name: "Call of Duty: WaW", Aliases: []string{"t4", "waw"}, DefaultGamePort: 28960, DefaultQueryPort: 28960, Protocol: "quake3", Supports: []string{"players"}},
	{Slug: "quake-live", Name: "Quake Live", Aliases: []string{"ql"}, AppID: 282440, DefaultGamePort: 27960, DefaultQueryPort: 27960, Protocol: "source", Supports: sourceSupports},
	{Slug: "wolfenstein-et", Name: "Wolfenstein: Enemy Territory", Aliases: []string{"et", "wet"}, DefaultGamePort: 27960, DefaultQueryPort: 27960, Protocol: "quake3", Supports: []string{"players"}},
	{
		Slug: "palworld", Name: "Palworld", Aliases: []string{"pw"},
		DefaultGamePort: 8211, DefaultQueryPort: 8211, Protocol: "eos",
		Supports: eosSupports, Notes: "Player names are opaque EOS IDs; may fail on shared IPs (15-session API limit)",
		ProtocolConfig: &protocol.EOSConfig{
			ClientID: "xyza78916PZ5DF0fAahu4tnrKKyFpqRE", ClientSecret: "j0NapLEPm3R3EOrlQiM8cRLKq3Rt02ZVVwT0SkZstSg",
			DeploymentID: "0a18471f93d448e2a1f60e47e03d3413", UseExternalAuth: true,
			Attributes: map[string]string{"name": "NAME_s", "map": "MAPNAME_s", "password": "ISPASSWORD_b", "version": "VERSION_s"},
		},
	},
	{Slug: "garrys-mod", Name: "Garry's Mod", Aliases: []string{"gmod"}, AppID: 4000, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{
		Slug: "ark-survival-ascended", Name: "ARK: Survival Ascended", Aliases: []string{"asa"},
		DefaultGamePort: 7777, DefaultQueryPort: 7777, Protocol: "eos",
		Supports: eosSupports, Notes: "Player names are opaque EOS IDs; may fail on shared IPs (15-session API limit)",
		ProtocolConfig: &protocol.EOSConfig{
			ClientID: "xyza7891muomRmynIIHaJB9COBKkwj6n", ClientSecret: "PP5UGxysEieNfSrEicaD1N2Bb3TdXuD7xHYcsdUHZ7s",
			DeploymentID: "ad9a8feffb3b4b2ca315546f038c3ae2", UseWildcard: true,
			Attributes: map[string]string{"name": "CUSTOMSERVERNAME_s", "map": "MAPNAME_s", "password": "SERVERPASSWORD_b", "version": "BUILDID_s"},
		},
	},
	{Slug: "ark-survival-evolved", Name: "ARK: Survival Evolved", Aliases: []string{"ark", "ase"}, AppID: 346110, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports, Notes: "Player list and max players may be inaccurate"},
	{Slug: "team-fortress-2", Name: "Team Fortress 2", Aliases: []string{"tf2"}, AppID: 440, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "dayz", Name: "DayZ", Aliases: []string{}, AppID: 221100, DefaultGamePort: 2302, DefaultQueryPort: 2303, Protocol: "source", Supports: sourceSupports},
	{Slug: "valheim", Name: "Valheim", Aliases: []string{}, AppID: 892970, DefaultGamePort: 2456, DefaultQueryPort: 2457, Protocol: "source", Supports: sourceSupports},
	{Slug: "7-days-to-die", Name: "7 Days to Die", Aliases: []string{"7dtd", "7d2d"}, AppID: 251570, DefaultGamePort: 26900, DefaultQueryPort: 26900, Protocol: "source", Supports: sourceSupports},
	{Slug: "left-4-dead-2", Name: "Left 4 Dead 2", Aliases: []string{"l4d2"}, AppID: 550, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "left-4-dead", Name: "Left 4 Dead", Aliases: []string{"l4d"}, AppID: 500, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "unturned", Name: "Unturned", Aliases: []string{}, AppID: 304930, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "hell-let-loose", Name: "Hell Let Loose", Aliases: []string{"hll"}, AppID: 686810, DefaultGamePort: 7777, DefaultQueryPort: 7778, Protocol: "source", Supports: sourceSupports},
	{Slug: "conan-exiles", Name: "Conan Exiles", Aliases: []string{"conan"}, AppID: 440900, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "enshrouded", Name: "Enshrouded", Aliases: []string{}, AppID: 1203620, DefaultGamePort: 15636, DefaultQueryPort: 15637, Protocol: "source", Supports: sourceSupports},
	{
		Slug: "squad", Name: "Squad", Aliases: []string{},
		DefaultGamePort: 7787, DefaultQueryPort: 7787, Protocol: "eos",
		Supports: eosSupports, Notes: "Player names are opaque EOS IDs; may fail on shared IPs (15-session API limit)",
		ProtocolConfig: &protocol.EOSConfig{
			ClientID: "xyza7891J7d3GU8ZIwCoC5xdBsdoqVWA", ClientSecret: "4SLVBqAm09q776SIlQRTD6moM/bnGAWhDSqOxJAIS0s",
			DeploymentID: "5dee4062a90b42cd98fcad618b6636c2", UseExternalAuth: true,
			Attributes: map[string]string{"name": "SERVERNAME_s", "password": "PASSWORD_b", "version": "GAMEVERSION_s"},
		},
	},
	{Slug: "project-zomboid", Name: "Project Zomboid", Aliases: []string{"pz"}, AppID: 108600, DefaultGamePort: 16261, DefaultQueryPort: 16261, Protocol: "source", Supports: sourceSupports},
	{Slug: "v-rising", Name: "V Rising", Aliases: []string{"vrising"}, AppID: 1604030, DefaultGamePort: 9876, DefaultQueryPort: 9877, Protocol: "source", Supports: sourceSupports},
	{Slug: "insurgency-sandstorm", Name: "Insurgency: Sandstorm", Aliases: []string{"inss"}, AppID: 581320, DefaultGamePort: 27102, DefaultQueryPort: 27131, Protocol: "source", Supports: sourceSupports},
	{Slug: "insurgency", Name: "Insurgency", Aliases: []string{"ins"}, AppID: 222880, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "the-forest", Name: "The Forest", Aliases: []string{"forest"}, AppID: 242760, DefaultGamePort: 27015, DefaultQueryPort: 27016, Protocol: "source", Supports: sourceSupports},
	{Slug: "sons-of-the-forest", Name: "Sons of the Forest", Aliases: []string{"sotf"}, AppID: 1326470, DefaultGamePort: 8766, DefaultQueryPort: 27016, Protocol: "source", Supports: sourceSupports},
	{Slug: "terraria", Name: "Terraria", Aliases: []string{"tshock"}, DefaultGamePort: 7777, DefaultQueryPort: 7878, Protocol: "tshock",
		Supports: []string{"players"}, Notes: "Requires TShock mod with RestApiEnabled — vanilla Terraria has no query protocol"},
	{Slug: "arma-3", Name: "Arma 3", Aliases: []string{"arma3"}, AppID: 107410, DefaultGamePort: 2302, DefaultQueryPort: 2303, Protocol: "source", Supports: sourceSupports},
	{Slug: "arma-2", Name: "Arma 2", Aliases: []string{"arma2"}, AppID: 33900, DefaultGamePort: 2302, DefaultQueryPort: 2303, Protocol: "source", Supports: sourceSupports},
	{Slug: "arma-reforger", Name: "Arma Reforger", Aliases: []string{"reforger"}, AppID: 1874880, DefaultGamePort: 2001, DefaultQueryPort: 17777, Protocol: "source", Supports: sourceSupports, Notes: "A2S must be enabled in server config (disabled by default)."},
	{Slug: "space-engineers", Name: "Space Engineers", Aliases: []string{"se"}, AppID: 244850, DefaultGamePort: 27016, DefaultQueryPort: 27016, Protocol: "source", Supports: sourceSupports},
	{Slug: "killing-floor-2", Name: "Killing Floor 2", Aliases: []string{"kf2"}, AppID: 232090, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "mordhau", Name: "Mordhau", Aliases: []string{}, AppID: 629760, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{
		Slug: "the-isle-evrima", Name: "The Isle: EVRIMA", Aliases: []string{"evrima"},
		DefaultGamePort: 7777, DefaultQueryPort: 7777, Protocol: "eos",
		Supports: eosSupports, Notes: "Player names are opaque EOS IDs; may fail on shared IPs (15-session API limit)",
		ProtocolConfig: &protocol.EOSConfig{
			ClientID: "xyza7891gk5PRo3J7G9puCJGFJjmEguW", ClientSecret: "pKWl6t5i9NJK8gTpVlAxzENZ65P8hYzodV8Dqe5Rlc8",
			DeploymentID: "6db6bea492f94b1bbdfcdfe3e4f898dc",
			Attributes:   map[string]string{"name": "SERVERNAME_s", "map": "MAP_NAME_s", "version": "SERVER_VERSION_s"},
		},
	},
	{Slug: "the-isle", Name: "The Isle", Aliases: []string{"isle"}, AppID: 376210, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "barotrauma", Name: "Barotrauma", Aliases: []string{}, AppID: 602960, DefaultGamePort: 27015, DefaultQueryPort: 27016, Protocol: "source", Supports: sourceSupports},
	{Slug: "no-more-room-in-hell", Name: "No More Room in Hell", Aliases: []string{"nmrih"}, AppID: 224260, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "day-of-infamy", Name: "Day of Infamy", Aliases: []string{"doi"}, AppID: 447820, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "natural-selection-2", Name: "Natural Selection 2", Aliases: []string{"ns2"}, AppID: 4920, DefaultGamePort: 27015, DefaultQueryPort: 27016, Protocol: "source", Supports: sourceSupports},
	{Slug: "black-mesa", Name: "Black Mesa", Aliases: []string{"bm", "bms"}, AppID: 362890, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "rising-storm-2", Name: "Rising Storm 2: Vietnam", Aliases: []string{"rs2"}, AppID: 418460, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "red-orchestra-2", Name: "Red Orchestra 2", Aliases: []string{"ro2"}, AppID: 35450, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "squad-44", Name: "Squad 44", Aliases: []string{"s44"}, AppID: 736220, DefaultGamePort: 7787, DefaultQueryPort: 27165, Protocol: "source", Supports: sourceSupports},
	{Slug: "beyond-the-wire", Name: "Beyond the Wire", Aliases: []string{"btw"}, AppID: 1058650, DefaultGamePort: 7787, DefaultQueryPort: 27165, Protocol: "source", Supports: sourceSupports},
	{Slug: "soulmask", Name: "Soulmask", Aliases: []string{}, AppID: 2646460, DefaultGamePort: 8777, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "the-front", Name: "The Front", Aliases: []string{}, AppID: 2285150, DefaultGamePort: 5001, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "myth-of-empires", Name: "Myth of Empires", Aliases: []string{"moe"}, AppID: 1371580, DefaultGamePort: 11888, DefaultQueryPort: 12888, Protocol: "source", Supports: sourceSupports},
	{Slug: "empyrion", Name: "Empyrion: Galactic Survival", Aliases: []string{"egs"}, AppID: 383120, DefaultGamePort: 30000, DefaultQueryPort: 30001, Protocol: "source", Supports: sourceSupports},
	{Slug: "starbound", Name: "Starbound", Aliases: []string{}, AppID: 211820, DefaultGamePort: 21025, DefaultQueryPort: 21025, Protocol: "source", Supports: sourceSupports},
	{Slug: "holdfast", Name: "Holdfast: Nations At War", Aliases: []string{"holdfast-naw"}, AppID: 589290, DefaultGamePort: 20100, DefaultQueryPort: 27000, Protocol: "source", Supports: sourceSupports},
	{Slug: "miscreated", Name: "Miscreated", Aliases: []string{}, AppID: 299740, DefaultGamePort: 64090, DefaultQueryPort: 64092, Protocol: "source", Supports: sourceSupports},
	{Slug: "hurtworld", Name: "Hurtworld", Aliases: []string{}, AppID: 393420, DefaultGamePort: 12871, DefaultQueryPort: 12881, Protocol: "source", Supports: sourceSupports},
	{Slug: "day-of-defeat-source", Name: "Day of Defeat: Source", Aliases: []string{"dods"}, AppID: 300, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "day-of-defeat", Name: "Day of Defeat", Aliases: []string{"dod"}, AppID: 30, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "sven-coop", Name: "Sven Co-op", Aliases: []string{"svencoop"}, AppID: 225840, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "half-life-2-deathmatch", Name: "Half-Life 2: Deathmatch", Aliases: []string{"hl2dm"}, AppID: 320, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "half-life-deathmatch-source", Name: "Half-Life Deathmatch: Source", Aliases: []string{"hldms"}, AppID: 360, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "half-life", Name: "Half-Life", Aliases: []string{"hl", "hl1"}, AppID: 70, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "half-life-opposing-force", Name: "Half-Life: Opposing Force", Aliases: []string{"hlof", "opfor"}, AppID: 50, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "atlas", Name: "Atlas", Aliases: []string{}, AppID: 834910, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "dark-and-light", Name: "Dark and Light", Aliases: []string{"dnl"}, AppID: 529180, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "pixark", Name: "PixARK", Aliases: []string{}, AppID: 593600, DefaultGamePort: 27015, DefaultQueryPort: 27016, Protocol: "source", Supports: sourceSupports},
	{Slug: "alien-swarm-reactive-drop", Name: "Alien Swarm: Reactive Drop", Aliases: []string{"asrd"}, AppID: 563560, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "fistful-of-frags", Name: "Fistful of Frags", Aliases: []string{"fof"}, AppID: 265630, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "zombie-panic-source", Name: "Zombie Panic Source", Aliases: []string{"zps"}, AppID: 17500, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "battalion-1944", Name: "Battalion 1944", Aliases: []string{"battalion"}, AppID: 489940, DefaultGamePort: 7777, DefaultQueryPort: 7780, Protocol: "source", Supports: sourceSupports},
	{Slug: "nuclear-dawn", Name: "Nuclear Dawn", Aliases: []string{"nd"}, AppID: 17710, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "brainbread-2", Name: "BrainBread 2", Aliases: []string{"bb2"}, AppID: 346330, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "neotokyo", Name: "NEOTOKYO", Aliases: []string{"nt"}, AppID: 244630, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "pirates-vikings-and-knights-ii", Name: "Pirates, Vikings, and Knights II", Aliases: []string{"pvkii"}, AppID: 17570, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "team-fortress-classic", Name: "Team Fortress Classic", Aliases: []string{"tfc"}, AppID: 20, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "deathmatch-classic", Name: "Deathmatch Classic", Aliases: []string{"dmc"}, AppID: 40, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
	{Slug: "ricochet", Name: "Ricochet", Aliases: []string{}, AppID: 60, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source", Supports: sourceSupports},
}

// Port indexes for fast lookup — built once at init.
var (
	queryPortIndex map[uint16][]*GameConfig
	gamePortIndex  map[uint16][]*GameConfig
)

func init() {
	queryPortIndex = make(map[uint16][]*GameConfig)
	gamePortIndex = make(map[uint16][]*GameConfig)
	for i := range gameRegistry {
		g := &gameRegistry[i]
		queryPortIndex[g.DefaultQueryPort] = append(queryPortIndex[g.DefaultQueryPort], g)
		gamePortIndex[g.DefaultGamePort] = append(gamePortIndex[g.DefaultGamePort], g)
	}
}

// GamesWithQueryPort returns games that use the given port as their default query port.
func GamesWithQueryPort(port uint16) []*GameConfig { return queryPortIndex[port] }

// GamesWithGamePort returns games that use the given port as their default game port.
func GamesWithGamePort(port uint16) []*GameConfig { return gamePortIndex[port] }

// SupportedGames returns all registered game configs.
func SupportedGames() []GameConfig {
	result := make([]GameConfig, len(gameRegistry))
	copy(result, gameRegistry)
	return result
}

// LookupGameByAppID finds a game by Steam AppID. Returns nil if not found.
func LookupGameByAppID(appID uint32) *GameConfig {
	if appID == 0 {
		return nil
	}
	for i := range gameRegistry {
		if gameRegistry[i].AppID == appID {
			return &gameRegistry[i]
		}
	}
	return nil
}

// LookupGame finds a game by slug or alias. Returns nil if not found.
func LookupGame(name string) *GameConfig {
	for i := range gameRegistry {
		g := &gameRegistry[i]
		if g.Slug == name {
			return g
		}
		for _, alias := range g.Aliases {
			if alias == name {
				return g
			}
		}
	}
	return nil
}
