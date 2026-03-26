package gjq

import (
	"testing"
)

func TestBuildCandidates(t *testing.T) {
	rust := Registry.Get("rust")
	if rust == nil {
		t.Fatal("rust game not found in registry")
	}
	mc := Registry.Get("minecraft-java")
	if mc == nil {
		t.Fatal("minecraft-java game not found in registry")
	}

	t.Run("direct with game", func(t *testing.T) {
		cs := buildCandidates(12345, rust, true, "")
		if len(cs) != 1 {
			t.Fatalf("got %d candidates, want 1", len(cs))
		}
		if cs[0].port != 12345 {
			t.Errorf("port = %d, want 12345", cs[0].port)
		}
		if cs[0].protocol != "source" {
			t.Errorf("protocol = %q, want source", cs[0].protocol)
		}
		if cs[0].priority != 0 {
			t.Errorf("priority = %d, want 0", cs[0].priority)
		}
	})

	t.Run("direct without game", func(t *testing.T) {
		cs := buildCandidates(27015, nil, true, "")
		if len(cs) == 0 {
			t.Fatal("got 0 candidates, want at least 1")
		}
		// All should be at the same port with priority 0
		for _, c := range cs {
			if c.port != 27015 {
				t.Errorf("port = %d, want 27015", c.port)
			}
			if c.priority != 0 {
				t.Errorf("priority = %d, want 0", c.priority)
			}
		}
		// Should not include EOS (requires game-specific credentials)
		for _, c := range cs {
			if c.protocol == "eos" {
				t.Error("direct without game should not include EOS protocol")
			}
		}
	})

	t.Run("game with default query port", func(t *testing.T) {
		cs := buildCandidates(rust.QueryPort(), rust, false, "")
		if len(cs) != 1 {
			t.Fatalf("got %d candidates, want 1", len(cs))
		}
		if cs[0].port != rust.QueryPort() {
			t.Errorf("port = %d, want %d", cs[0].port, rust.QueryPort())
		}
	})

	t.Run("game with game port differs from query port", func(t *testing.T) {
		cs := buildCandidates(rust.GamePort(), rust, false, "")
		if len(cs) < 2 {
			t.Fatalf("got %d candidates, want at least 2", len(cs))
		}
		// First candidate should be the default query port at priority 0
		if cs[0].port != rust.QueryPort() || cs[0].priority != 0 {
			t.Errorf("first candidate: port=%d priority=%d, want port=%d priority=0", cs[0].port, cs[0].priority, rust.QueryPort())
		}
	})

	t.Run("game with same game and query port", func(t *testing.T) {
		cs := buildCandidates(mc.QueryPort(), mc, false, "")
		if len(cs) != 1 {
			t.Fatalf("got %d candidates, want 1", len(cs))
		}
	})

	t.Run("auto-detect includes user port", func(t *testing.T) {
		cs := buildCandidates(27015, nil, false, "")
		hasUserPort := false
		for _, c := range cs {
			if c.port == 27015 && c.priority == 0 {
				hasUserPort = true
				break
			}
		}
		if !hasUserPort {
			t.Error("auto-detect candidates missing user port at priority 0")
		}
	})
}

func TestBuildCandidatesWithGameOffsets(t *testing.T) {
	// Rust: game=28015, query=28017, offset=+2
	rust := Registry.Get("rust")
	if rust == nil {
		t.Fatal("rust not found")
	}

	t.Run("arbitrary port derives offsets both directions", func(t *testing.T) {
		// Port 30000 (not default game or query) should try:
		// priority 0: 30000, priority 1: 30000+2=30002 and 30000-2=29998
		cs := buildCandidatesWithGame(30000, rust)
		if len(cs) != 3 {
			t.Fatalf("got %d candidates, want 3", len(cs))
		}
		if cs[0].port != 30000 || cs[0].priority != 0 {
			t.Errorf("cs[0] = {port:%d, pri:%d}, want {port:30000, pri:0}", cs[0].port, cs[0].priority)
		}
		// Check that offset-derived ports exist at priority 1
		derivedPorts := map[uint16]bool{}
		for _, c := range cs[1:] {
			derivedPorts[c.port] = true
			if c.priority != 1 {
				t.Errorf("derived candidate port %d has priority %d, want 1", c.port, c.priority)
			}
		}
		if !derivedPorts[30002] {
			t.Error("missing derived port 30002 (+2 offset)")
		}
		if !derivedPorts[29998] {
			t.Error("missing derived port 29998 (-2 offset)")
		}
	})

	t.Run("low port skips invalid negative offset", func(t *testing.T) {
		// Port 1 with offset +2: 1+2=3 valid, 1-2=-1 invalid
		cs := buildCandidatesWithGame(1, rust)
		for _, c := range cs {
			if c.port == 0 {
				t.Error("candidate with port 0 should not exist")
			}
		}
	})

	t.Run("same game and query port skips offsets", func(t *testing.T) {
		mc := Registry.Get("minecraft-java")
		if mc == nil {
			t.Fatal("minecraft-java not found")
		}
		// game=25565, query=25565 — no offset to derive
		cs := buildCandidatesWithGame(30000, mc)
		if len(cs) != 1 {
			t.Fatalf("got %d candidates, want 1 (no offsets when game==query port)", len(cs))
		}
	})
}
