package gsq

type GameConfig struct {
	Slug             string
	Aliases          []string
	AppID            uint32
	DefaultGamePort  uint16
	DefaultQueryPort uint16
	Protocol         string
}

var gameRegistry = []GameConfig{
	// Minecraft
	{Slug: "minecraft", Aliases: []string{"mc"}, DefaultGamePort: 25565, DefaultQueryPort: 25565, Protocol: "minecraft"},

	// Source engine (A2S) — roughly ordered by popularity
	{Slug: "counter-strike-2", Aliases: []string{"cs2"}, AppID: 730, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "counter-strike", Aliases: []string{"cs16", "cs"}, AppID: 10, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "counter-strike-source", Aliases: []string{"css"}, AppID: 240, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "counter-strike-go", Aliases: []string{"csgo"}, AppID: 730, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "rust", Aliases: []string{}, AppID: 252490, DefaultGamePort: 28015, DefaultQueryPort: 28017, Protocol: "source"},
	{Slug: "ark-survival-evolved", Aliases: []string{"ark", "ase"}, AppID: 346110, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "valheim", Aliases: []string{}, AppID: 892970, DefaultGamePort: 2456, DefaultQueryPort: 2457, Protocol: "source"},
	{Slug: "dayz", Aliases: []string{}, AppID: 221100, DefaultGamePort: 2302, DefaultQueryPort: 2303, Protocol: "source"},
	{Slug: "garrys-mod", Aliases: []string{"gmod"}, AppID: 4000, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "team-fortress-2", Aliases: []string{"tf2"}, AppID: 440, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "conan-exiles", Aliases: []string{"conan"}, AppID: 440900, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "7-days-to-die", Aliases: []string{"7dtd"}, AppID: 251570, DefaultGamePort: 26900, DefaultQueryPort: 26900, Protocol: "source"},
	{Slug: "hell-let-loose", Aliases: []string{"hll"}, AppID: 686810, DefaultGamePort: 7777, DefaultQueryPort: 7778, Protocol: "source"},
	{Slug: "enshrouded", Aliases: []string{}, AppID: 1203620, DefaultGamePort: 15636, DefaultQueryPort: 15637, Protocol: "source"},
	{Slug: "v-rising", Aliases: []string{"vrising"}, AppID: 1604030, DefaultGamePort: 9876, DefaultQueryPort: 9877, Protocol: "source"},
	{Slug: "unturned", Aliases: []string{}, AppID: 304930, DefaultGamePort: 27015, DefaultQueryPort: 27016, Protocol: "source"},
	{Slug: "project-zomboid", Aliases: []string{"pz"}, AppID: 108600, DefaultGamePort: 16261, DefaultQueryPort: 16261, Protocol: "source"},
	{Slug: "space-engineers", Aliases: []string{"se"}, AppID: 244850, DefaultGamePort: 27016, DefaultQueryPort: 27016, Protocol: "source"},
	{Slug: "insurgency-sandstorm", Aliases: []string{"inss"}, AppID: 581320, DefaultGamePort: 27102, DefaultQueryPort: 27131, Protocol: "source"},
	{Slug: "insurgency", Aliases: []string{"ins"}, AppID: 222880, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "mordhau", Aliases: []string{}, AppID: 629760, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "killing-floor-2", Aliases: []string{"kf2"}, AppID: 232090, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "barotrauma", Aliases: []string{}, AppID: 602960, DefaultGamePort: 27015, DefaultQueryPort: 27016, Protocol: "source"},
	{Slug: "arma-3", Aliases: []string{"arma3"}, AppID: 107410, DefaultGamePort: 2302, DefaultQueryPort: 2303, Protocol: "source"},
	{Slug: "arma-2", Aliases: []string{"arma2"}, AppID: 33900, DefaultGamePort: 2302, DefaultQueryPort: 2303, Protocol: "source"},
	{Slug: "squad-44", Aliases: []string{"s44"}, AppID: 736220, DefaultGamePort: 7787, DefaultQueryPort: 27165, Protocol: "source"},
	{Slug: "beyond-the-wire", Aliases: []string{"btw"}, AppID: 1058650, DefaultGamePort: 7787, DefaultQueryPort: 27165, Protocol: "source"},
	{Slug: "rising-storm-2", Aliases: []string{"rs2"}, AppID: 418460, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "the-front", Aliases: []string{}, AppID: 2285150, DefaultGamePort: 5001, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "soulmask", Aliases: []string{}, AppID: 2646460, DefaultGamePort: 8777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "myth-of-empires", Aliases: []string{"moe"}, AppID: 1371580, DefaultGamePort: 11888, DefaultQueryPort: 12888, Protocol: "source"},
	{Slug: "atlas", Aliases: []string{}, AppID: 834910, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "dark-and-light", Aliases: []string{"dnl"}, AppID: 529180, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "pixark", Aliases: []string{}, AppID: 593600, DefaultGamePort: 27015, DefaultQueryPort: 27016, Protocol: "source"},
	{Slug: "battalion-1944", Aliases: []string{"battalion"}, AppID: 489940, DefaultGamePort: 7777, DefaultQueryPort: 7780, Protocol: "source"},
	{Slug: "starbound", Aliases: []string{}, AppID: 211820, DefaultGamePort: 21025, DefaultQueryPort: 21025, Protocol: "source"},
}

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
