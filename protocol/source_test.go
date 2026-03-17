package protocol

import (
	"encoding/binary"
	"math"
	"testing"
)

// buildInfoResponse constructs a minimal A2S_INFO response payload (after type byte).
func buildInfoResponse(name, mapName, folder, game string, appID uint16, players, maxPlayers, bots uint8, serverType, env, vis, vac uint8, version string) []byte {
	var buf []byte
	buf = append(buf, 0x11) // protocol version
	buf = append(buf, []byte(name)...)
	buf = append(buf, 0x00)
	buf = append(buf, []byte(mapName)...)
	buf = append(buf, 0x00)
	buf = append(buf, []byte(folder)...)
	buf = append(buf, 0x00)
	buf = append(buf, []byte(game)...)
	buf = append(buf, 0x00)

	// Fixed fields
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, appID)
	buf = append(buf, b...)
	buf = append(buf, players, maxPlayers, bots, serverType, env, vis, vac)

	buf = append(buf, []byte(version)...)
	buf = append(buf, 0x00)

	return buf
}

func TestParseInfoResponse(t *testing.T) {
	t.Run("valid response", func(t *testing.T) {
		data := buildInfoResponse(
			"Test Server", "de_dust2", "csgo", "Counter-Strike 2",
			730, 16, 32, 0, 'd', 'l', 0, 1, "1.38.7.1",
		)

		info, err := parseInfoResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Name != "Test Server" {
			t.Errorf("Name = %q, want %q", info.Name, "Test Server")
		}
		if info.Map != "de_dust2" {
			t.Errorf("Map = %q, want %q", info.Map, "de_dust2")
		}
		if info.Game != "Counter-Strike 2" {
			t.Errorf("Game = %q, want %q", info.Game, "Counter-Strike 2")
		}
		if info.Players != 16 {
			t.Errorf("Players = %d, want 16", info.Players)
		}
		if info.MaxPlayers != 32 {
			t.Errorf("MaxPlayers = %d, want 32", info.MaxPlayers)
		}
		if info.ServerType != "dedicated" {
			t.Errorf("ServerType = %q, want %q", info.ServerType, "dedicated")
		}
		if info.Environment != "linux" {
			t.Errorf("Environment = %q, want %q", info.Environment, "linux")
		}
		if info.Visibility != "public" {
			t.Errorf("Visibility = %q, want %q", info.Visibility, "public")
		}
		if !info.VAC {
			t.Error("VAC = false, want true")
		}
		if info.Version != "1.38.7.1" {
			t.Errorf("Version = %q, want %q", info.Version, "1.38.7.1")
		}
		if info.AppID != 730 {
			t.Errorf("AppID = %d, want 730", info.AppID)
		}
		if info.Extra["folder"] != "csgo" {
			t.Errorf("Extra[folder] = %v, want %q", info.Extra["folder"], "csgo")
		}
	})

	t.Run("empty folder omitted from extra", func(t *testing.T) {
		data := buildInfoResponse("S", "m", "", "g", 0, 0, 0, 0, 'd', 'l', 0, 0, "1.0")
		info, err := parseInfoResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := info.Extra["folder"]; ok {
			t.Error("empty folder should not be in Extra")
		}
	})

	t.Run("truncated input", func(t *testing.T) {
		_, err := parseInfoResponse([]byte{0x11})
		if err == nil {
			t.Fatal("expected error for truncated input")
		}
	})
}

// appendEDF appends an EDF byte and its associated fields to an info response.
func appendEDF(base []byte, edf byte, gamePort uint16, steamID uint64, specPort uint16, specName string, keywords string, gameID uint64) []byte {
	buf := append([]byte{}, base...)
	buf = append(buf, edf)
	b := make([]byte, 8)
	if edf&0x80 != 0 {
		binary.LittleEndian.PutUint16(b[:2], gamePort)
		buf = append(buf, b[:2]...)
	}
	if edf&0x10 != 0 {
		binary.LittleEndian.PutUint64(b, steamID)
		buf = append(buf, b...)
	}
	if edf&0x40 != 0 {
		binary.LittleEndian.PutUint16(b[:2], specPort)
		buf = append(buf, b[:2]...)
		buf = append(buf, []byte(specName)...)
		buf = append(buf, 0x00)
	}
	if edf&0x20 != 0 {
		buf = append(buf, []byte(keywords)...)
		buf = append(buf, 0x00)
	}
	if edf&0x01 != 0 {
		binary.LittleEndian.PutUint64(b, gameID)
		buf = append(buf, b...)
	}
	return buf
}

func TestParseInfoResponseEDF(t *testing.T) {
	base := buildInfoResponse("Server", "de_dust2", "csgo", "CS2", 730, 16, 32, 0, 'd', 'l', 0, 1, "1.38.7.1")

	t.Run("keywords and steamID", func(t *testing.T) {
		data := appendEDF(base, 0x30, 0, 76561198000000000, 0, "", "valve,secure", 0)
		info, err := parseInfoResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Keywords != "valve,secure" {
			t.Errorf("Keywords = %q, want %q", info.Keywords, "valve,secure")
		}
		sid, ok := info.Extra["steamId"].(uint64)
		if !ok {
			t.Fatalf("Extra[steamId] not uint64, got %T", info.Extra["steamId"])
		}
		if sid != 76561198000000000 {
			t.Errorf("steamId = %d, want 76561198000000000", sid)
		}
	})

	t.Run("gameID overrides uint16 AppID", func(t *testing.T) {
		// uint16 AppID=730, but gameID=730 in 64-bit field should also produce 730
		// Use a different gameID to verify it overrides
		data := appendEDF(base, 0x01, 0, 0, 0, "", "", 12345678)
		info, err := parseInfoResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// gameID & 0xFFFFFF = 12345678 & 0xFFFFFF = 6078510
		want := uint32(12345678 & 0xFFFFFF)
		if info.AppID != want {
			t.Errorf("AppID = %d, want %d (from gameID)", info.AppID, want)
		}
	})

	t.Run("game port flag skipped correctly", func(t *testing.T) {
		// EDF 0xB0 = game port (0x80) + steamID (0x10) + keywords (0x20)
		data := appendEDF(base, 0xB0, 27016, 12345, 0, "", "test_kw", 0)
		info, err := parseInfoResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify steamID and keywords parsed correctly despite game port being before them
		if info.Keywords != "test_kw" {
			t.Errorf("Keywords = %q, want %q", info.Keywords, "test_kw")
		}
		sid, ok := info.Extra["steamId"].(uint64)
		if !ok || sid != 12345 {
			t.Errorf("steamId = %v, want 12345", info.Extra["steamId"])
		}
	})

	t.Run("spectator info skipped correctly", func(t *testing.T) {
		// EDF 0x60 = spectator (0x40) + keywords (0x20)
		data := appendEDF(base, 0x60, 0, 0, 27020, "SourceTV", "after_spec", 0)
		info, err := parseInfoResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Keywords != "after_spec" {
			t.Errorf("Keywords = %q, want %q (after skipping spectator fields)", info.Keywords, "after_spec")
		}
	})

	t.Run("no EDF byte", func(t *testing.T) {
		info, err := parseInfoResponse(base)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should use uint16 AppID fallback
		if info.AppID != 730 {
			t.Errorf("AppID = %d, want 730 (uint16 fallback)", info.AppID)
		}
		if info.Keywords != "" {
			t.Errorf("Keywords = %q, want empty", info.Keywords)
		}
	})

	t.Run("all flags set", func(t *testing.T) {
		data := appendEDF(base, 0xF1, 27016, 76561198000000000, 27020, "SourceTV", "all,flags", 730)
		info, err := parseInfoResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Keywords != "all,flags" {
			t.Errorf("Keywords = %q, want %q", info.Keywords, "all,flags")
		}
		if info.Extra["steamId"] != uint64(76561198000000000) {
			t.Errorf("steamId = %v, want 76561198000000000", info.Extra["steamId"])
		}
		if info.AppID != uint32(730&0xFFFFFF) {
			t.Errorf("AppID = %d, want %d", info.AppID, 730&0xFFFFFF)
		}
	})
}

// buildPlayerResponse constructs an A2S_PLAYER response payload (after type byte).
func buildPlayerResponse(players []struct {
	name     string
	score    int32
	duration float32
}) []byte {
	var buf []byte
	buf = append(buf, uint8(len(players)))
	for i, p := range players {
		buf = append(buf, uint8(i)) // index
		buf = append(buf, []byte(p.name)...)
		buf = append(buf, 0x00)
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(p.score))
		buf = append(buf, b...)
		bits := math.Float32bits(p.duration)
		binary.LittleEndian.PutUint32(b, bits)
		buf = append(buf, b...)
	}
	return buf
}

func TestParsePlayerResponse(t *testing.T) {
	t.Run("valid players", func(t *testing.T) {
		input := []struct {
			name     string
			score    int32
			duration float32
		}{
			{"Alice", 10, 3600.0},
			{"Bob", 5, 1800.0},
		}
		data := buildPlayerResponse(input)

		players, err := parsePlayerResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(players) != 2 {
			t.Fatalf("got %d players, want 2", len(players))
		}
		if players[0].Name != "Alice" {
			t.Errorf("players[0].Name = %q, want %q", players[0].Name, "Alice")
		}
		if players[0].Score != 10 {
			t.Errorf("players[0].Score = %d, want 10", players[0].Score)
		}
		if players[1].Name != "Bob" {
			t.Errorf("players[1].Name = %q, want %q", players[1].Name, "Bob")
		}
	})

	t.Run("empty player list", func(t *testing.T) {
		data := []byte{0} // 0 players
		players, err := parsePlayerResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(players) != 0 {
			t.Errorf("got %d players, want 0", len(players))
		}
	})

	t.Run("truncated response returns partial list", func(t *testing.T) {
		// Claim 2 players but only include data for 1
		input := []struct {
			name     string
			score    int32
			duration float32
		}{
			{"Alice", 10, 3600.0},
		}
		data := buildPlayerResponse(input)
		data[0] = 2 // lie about count

		players, err := parsePlayerResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(players) != 1 {
			t.Errorf("got %d players, want 1 (partial)", len(players))
		}
	})
}

// buildRulesResponse constructs an A2S_RULES response payload (after type byte).
func buildRulesResponse(rules map[string]string) []byte {
	var buf []byte
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(len(rules)))
	buf = append(buf, b...)
	for k, v := range rules {
		buf = append(buf, []byte(k)...)
		buf = append(buf, 0x00)
		buf = append(buf, []byte(v)...)
		buf = append(buf, 0x00)
	}
	return buf
}

func TestParseRulesResponse(t *testing.T) {
	t.Run("valid rules", func(t *testing.T) {
		input := map[string]string{
			"mp_maxrounds": "30",
			"sv_cheats":    "0",
		}
		data := buildRulesResponse(input)

		rules, err := parseRulesResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rules) != 2 {
			t.Fatalf("got %d rules, want 2", len(rules))
		}
		if rules["mp_maxrounds"] != "30" {
			t.Errorf("mp_maxrounds = %q, want %q", rules["mp_maxrounds"], "30")
		}
		if rules["sv_cheats"] != "0" {
			t.Errorf("sv_cheats = %q, want %q", rules["sv_cheats"], "0")
		}
	})

	t.Run("empty rules", func(t *testing.T) {
		data := buildRulesResponse(map[string]string{})
		rules, err := parseRulesResponse(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rules) != 0 {
			t.Errorf("got %d rules, want 0", len(rules))
		}
	})

	t.Run("truncated response returns partial", func(t *testing.T) {
		// Build 1 rule but claim 2
		var buf []byte
		b := make([]byte, 2)
		binary.LittleEndian.PutUint16(b, 2) // claim 2 rules
		buf = append(buf, b...)
		buf = append(buf, []byte("key")...)
		buf = append(buf, 0x00)
		buf = append(buf, []byte("val")...)
		buf = append(buf, 0x00)

		rules, err := parseRulesResponse(buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rules) != 1 {
			t.Errorf("got %d rules, want 1 (partial)", len(rules))
		}
	})
}
