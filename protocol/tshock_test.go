package protocol

import (
	"testing"
	"time"
)

func TestSplitPlayers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"multiple players", "Alice, Bob, Charlie", []string{"Alice", "Bob", "Charlie"}},
		{"whitespace trimmed", "  Alice , Bob ", []string{"Alice", "Bob"}},
		{"empty string", "", nil},
		{"single player", "Solo", []string{"Solo"}},
		{"empty entries skipped", ", , ,", nil},
		{"trailing comma", "Alice, Bob,", []string{"Alice", "Bob"}},
		{"only spaces", "   ", nil},
		{"tabs trimmed", "\tAlice\t, Bob", []string{"Alice", "Bob"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPlayers(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitPlayers(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitPlayers(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMapTshockStatus(t *testing.T) {
	t.Run("full status", func(t *testing.T) {
		status := tshockStatusResponse{
			Name:           "My Terraria Server",
			Port:           7777,
			PlayerCount:    5,
			MaxPlayers:     16,
			World:          "LargeWorld",
			Players:        "Alice, Bob, Charlie",
			ServerVersion:  "v1.4.4.9",
			ServerPassword: false,
			TShockVersion:  "5.2.0",
			Uptime:         "1d 2h 30m",
		}

		info := mapTshockStatus(status, 7878, 10*time.Millisecond, true)

		if info.Name != "My Terraria Server" {
			t.Errorf("Name = %q, want %q", info.Name, "My Terraria Server")
		}
		if info.Map != "LargeWorld" {
			t.Errorf("Map = %q, want %q", info.Map, "LargeWorld")
		}
		if info.Players != 5 {
			t.Errorf("Players = %d, want 5", info.Players)
		}
		if info.MaxPlayers != 16 {
			t.Errorf("MaxPlayers = %d, want 16", info.MaxPlayers)
		}
		if info.Version != "v1.4.4.9" {
			t.Errorf("Version = %q, want %q", info.Version, "v1.4.4.9")
		}
		if info.Visibility != "public" {
			t.Errorf("Visibility = %q, want %q", info.Visibility, "public")
		}
		if info.GamePort != 7777 {
			t.Errorf("GamePort = %d, want 7777", info.GamePort)
		}
		if info.QueryPort != 7878 {
			t.Errorf("QueryPort = %d, want 7878", info.QueryPort)
		}
		if info.Protocol != "tshock" {
			t.Errorf("Protocol = %q, want %q", info.Protocol, "tshock")
		}
		if info.Extra["tshockVersion"] != "5.2.0" {
			t.Errorf("Extra[tshockVersion] = %v, want %q", info.Extra["tshockVersion"], "5.2.0")
		}
		if info.Extra["uptime"] != "1d 2h 30m" {
			t.Errorf("Extra[uptime] = %v, want %q", info.Extra["uptime"], "1d 2h 30m")
		}
		if len(info.PlayerList) != 3 {
			t.Fatalf("PlayerList len = %d, want 3", len(info.PlayerList))
		}
		if info.PlayerList[0].Name != "Alice" {
			t.Errorf("PlayerList[0].Name = %q, want %q", info.PlayerList[0].Name, "Alice")
		}
	})

	t.Run("empty name falls back to world", func(t *testing.T) {
		status := tshockStatusResponse{
			Name:  "",
			World: "FallbackWorld",
		}
		info := mapTshockStatus(status, 7878, 0, false)
		if info.Name != "FallbackWorld" {
			t.Errorf("Name = %q, want %q (fallback to World)", info.Name, "FallbackWorld")
		}
	})

	t.Run("password protected", func(t *testing.T) {
		status := tshockStatusResponse{
			Name:           "Private Server",
			ServerPassword: true,
		}
		info := mapTshockStatus(status, 7878, 0, false)
		if info.Visibility != "private" {
			t.Errorf("Visibility = %q, want %q", info.Visibility, "private")
		}
	})

	t.Run("players not fetched when disabled", func(t *testing.T) {
		status := tshockStatusResponse{
			Players: "Alice, Bob",
		}
		info := mapTshockStatus(status, 7878, 0, false)
		if len(info.PlayerList) != 0 {
			t.Errorf("PlayerList len = %d, want 0 (players disabled)", len(info.PlayerList))
		}
	})

	t.Run("no extra when versions empty", func(t *testing.T) {
		status := tshockStatusResponse{Name: "S"}
		info := mapTshockStatus(status, 7878, 0, false)
		if info.Extra != nil {
			t.Errorf("Extra = %v, want nil when no tshock version or uptime", info.Extra)
		}
	})
}
