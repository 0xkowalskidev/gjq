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
	{Slug: "counter-strike-2", Aliases: []string{"cs2"}, AppID: 730, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "team-fortress-2", Aliases: []string{"tf2"}, AppID: 440, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "counter-strike-go", Aliases: []string{"csgo"}, AppID: 730, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "garrys-mod", Aliases: []string{"gmod"}, AppID: 4000, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "rust", Aliases: []string{}, AppID: 252490, DefaultGamePort: 28015, DefaultQueryPort: 28017, Protocol: "source"},
	{Slug: "ark-survival-evolved", Aliases: []string{"ark"}, AppID: 346110, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "dayz", Aliases: []string{}, AppID: 221100, DefaultGamePort: 2302, DefaultQueryPort: 2303, Protocol: "source"},
	{Slug: "left-4-dead-2", Aliases: []string{"l4d2"}, AppID: 550, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "minecraft", Aliases: []string{"mc"}, DefaultGamePort: 25565, DefaultQueryPort: 25565, Protocol: "minecraft"},
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
