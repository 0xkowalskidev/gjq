package gjq

import (
	"testing"

	"github.com/0xkowalskidev/gjq/protocol"
)

func TestLookupGame(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSlug string
		wantNil  bool
	}{
		{"slug lookup", "minecraft-java", "minecraft-java", false},
		{"alias lookup", "mc", "minecraft-java", false},
		{"another alias", "cs2", "counter-strike-2", false},
		{"unknown name", "nonexistent-game", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LookupGame(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("LookupGame(%q) = %+v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("LookupGame(%q) = nil, want slug %q", tt.input, tt.wantSlug)
			}
			if got.Slug != tt.wantSlug {
				t.Errorf("LookupGame(%q).Slug = %q, want %q", tt.input, got.Slug, tt.wantSlug)
			}
		})
	}
}

func TestLookupGameByAppID(t *testing.T) {
	tests := []struct {
		name     string
		appID    uint32
		wantSlug string
		wantNil  bool
	}{
		{"valid AppID", 730, "counter-strike-2", false},
		{"zero AppID", 0, "", true},
		{"unknown AppID", 99999, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LookupGameByAppID(tt.appID)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("LookupGameByAppID(%d) = %+v, want nil", tt.appID, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("LookupGameByAppID(%d) = nil, want slug %q", tt.appID, tt.wantSlug)
			}
			if got.Slug != tt.wantSlug {
				t.Errorf("LookupGameByAppID(%d).Slug = %q, want %q", tt.appID, got.Slug, tt.wantSlug)
			}
		})
	}
}

func TestGamesWithQueryPort(t *testing.T) {
	tests := []struct {
		name     string
		port     uint16
		wantMin  int // at least this many games
		wantNone bool
	}{
		{"common source port", 27015, 1, false},
		{"minecraft java port", 25565, 1, false},
		{"unused port", 1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GamesWithQueryPort(tt.port)
			if tt.wantNone && len(got) > 0 {
				t.Fatalf("GamesWithQueryPort(%d) returned %d games, want 0", tt.port, len(got))
			}
			if !tt.wantNone && len(got) < tt.wantMin {
				t.Fatalf("GamesWithQueryPort(%d) returned %d games, want at least %d", tt.port, len(got), tt.wantMin)
			}
		})
	}
}

func TestGamesWithGamePort(t *testing.T) {
	tests := []struct {
		name     string
		port     uint16
		wantMin  int
		wantNone bool
	}{
		{"common source port", 27015, 1, false},
		{"ark/isle game port", 7777, 1, false},
		{"unused port", 1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GamesWithGamePort(tt.port)
			if tt.wantNone && len(got) > 0 {
				t.Fatalf("GamesWithGamePort(%d) returned %d games, want 0", tt.port, len(got))
			}
			if !tt.wantNone && len(got) < tt.wantMin {
				t.Fatalf("GamesWithGamePort(%d) returned %d games, want at least %d", tt.port, len(got), tt.wantMin)
			}
		})
	}
}

func TestHasSupport(t *testing.T) {
	cs2 := LookupGame("cs2")
	if cs2 == nil {
		t.Fatal("cs2 not found")
	}
	tf2 := LookupGame("tf2")
	if tf2 == nil {
		t.Fatal("tf2 not found")
	}
	bedrock := LookupGame("minecraft-bedrock")
	if bedrock == nil {
		t.Fatal("minecraft-bedrock not found")
	}

	if !cs2.HasSupport("players") {
		t.Error("cs2 should support players")
	}
	if cs2.HasSupport("rules") {
		t.Error("cs2 should not support rules")
	}
	if !tf2.HasSupport("rules") {
		t.Error("tf2 should support rules")
	}
	if bedrock.HasSupport("players") {
		t.Error("bedrock should not support players")
	}
	if bedrock.HasSupport("nonexistent") {
		t.Error("bedrock should not support nonexistent feature")
	}
}

func TestRegistryConsistency(t *testing.T) {
	games := SupportedGames()

	slugs := make(map[string]bool)
	aliases := make(map[string]string) // alias → slug that owns it

	for _, g := range games {
		if g.Slug == "" {
			t.Error("game with empty slug found")
			continue
		}
		if g.Protocol == "" {
			t.Errorf("game %q has empty protocol", g.Slug)
		}
		if slugs[g.Slug] {
			t.Errorf("duplicate slug: %q", g.Slug)
		}
		slugs[g.Slug] = true

		for _, alias := range g.Aliases {
			if owner, ok := aliases[alias]; ok {
				t.Errorf("duplicate alias %q: used by both %q and %q", alias, owner, g.Slug)
			}
			aliases[alias] = g.Slug
		}

		// Verify protocol is actually registered
		if _, err := protocol.Get(g.Protocol); err != nil {
			t.Errorf("game %q uses unregistered protocol %q", g.Slug, g.Protocol)
		}
	}
}
