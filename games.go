package gsq

type GameConfig struct {
	Slug             string
	Aliases          []string
	DefaultGamePort  uint16
	DefaultQueryPort uint16
	Protocol         string
}

var gameRegistry = []GameConfig{
	{Slug: "counter-strike-2", Aliases: []string{"cs2"}, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "team-fortress-2", Aliases: []string{"tf2"}, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "counter-strike-go", Aliases: []string{"csgo"}, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "garrys-mod", Aliases: []string{"gmod"}, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "rust", Aliases: []string{}, DefaultGamePort: 28015, DefaultQueryPort: 28017, Protocol: "source"},
	{Slug: "ark-survival-evolved", Aliases: []string{"ark"}, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "dayz", Aliases: []string{}, DefaultGamePort: 2302, DefaultQueryPort: 2303, Protocol: "source"},
	{Slug: "left-4-dead-2", Aliases: []string{"l4d2"}, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "minecraft", Aliases: []string{"mc"}, DefaultGamePort: 25565, DefaultQueryPort: 25565, Protocol: "minecraft"},
}

// SupportedGames returns all registered game configs.
func SupportedGames() []GameConfig {
	result := make([]GameConfig, len(gameRegistry))
	copy(result, gameRegistry)
	return result
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
