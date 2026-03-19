package protocol

import (
	"testing"
	"time"
)

func TestParseQ3InfoString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{"basic", `\key1\val1\key2\val2`, map[string]string{"key1": "val1", "key2": "val2"}},
		{"single pair", `\hostname\My Server`, map[string]string{"hostname": "My Server"}},
		{"odd count", `\key1\val1\key2`, map[string]string{"key1": "val1"}},
		{"empty", "", map[string]string{}},
		{"no leading slash", `key1\val1`, map[string]string{"key1": "val1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseQ3InfoString(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d pairs, want %d: %v", len(got), len(tt.want), got)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("key %q = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestParseQ3Players(t *testing.T) {
	t.Run("multiple players", func(t *testing.T) {
		input := "15 40 \"PlayerOne\"\n-2 0 \"Bot_Grunt\"\n8 65 \"Noob\"\n"
		players := parseQ3Players(input)

		if len(players) != 3 {
			t.Fatalf("got %d players, want 3", len(players))
		}

		if players[0].Name != "PlayerOne" {
			t.Errorf("player 0 name = %q, want %q", players[0].Name, "PlayerOne")
		}
		if players[0].Score != 15 {
			t.Errorf("player 0 score = %d, want 15", players[0].Score)
		}
		if players[0].Duration.Duration != 40*time.Millisecond {
			t.Errorf("player 0 ping = %v, want 40ms", players[0].Duration.Duration)
		}

		if players[1].Name != "Bot_Grunt" {
			t.Errorf("player 1 name = %q, want %q", players[1].Name, "Bot_Grunt")
		}
		if players[1].Score != -2 {
			t.Errorf("player 1 score = %d, want -2", players[1].Score)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		players := parseQ3Players("")
		if len(players) != 0 {
			t.Errorf("got %d players, want 0", len(players))
		}
	})

	t.Run("trailing newlines", func(t *testing.T) {
		players := parseQ3Players("5 30 \"Test\"\n\n\n")
		if len(players) != 1 {
			t.Errorf("got %d players, want 1", len(players))
		}
	})
}

func TestParseQ3StatusResponse(t *testing.T) {
	t.Run("full response", func(t *testing.T) {
		data := []byte("\xFF\xFF\xFF\xFFstatusResponse\n\\sv_hostname\\My Server\\mapname\\q3dm6\\gametype\\0\\sv_maxclients\\16\\clients\\3\\g_needpass\\0\\game\\cpma\n15 40 \"PlayerOne\"\n-2 0 \"Bot\"\n8 65 \"Noob\"\n")

		info, err := parseQ3StatusResponse(data, 27960)
		if err != nil {
			t.Fatal(err)
		}

		if info.Name != "My Server" {
			t.Errorf("Name = %q, want %q", info.Name, "My Server")
		}
		if info.Map != "q3dm6" {
			t.Errorf("Map = %q, want %q", info.Map, "q3dm6")
		}
		if info.GameMode != "0" {
			t.Errorf("GameMode = %q, want %q", info.GameMode, "0")
		}
		if info.MaxPlayers != 16 {
			t.Errorf("MaxPlayers = %d, want 16", info.MaxPlayers)
		}
		if info.Players != 3 {
			t.Errorf("Players = %d, want 3", info.Players)
		}
		if info.Visibility != "public" {
			t.Errorf("Visibility = %q, want %q", info.Visibility, "public")
		}
		if info.GamePort != 27960 {
			t.Errorf("GamePort = %d, want 27960", info.GamePort)
		}
		if info.Extra["mod"] != "cpma" {
			t.Errorf("Extra[mod] = %v, want %q", info.Extra["mod"], "cpma")
		}
		if len(info.PlayerList) != 3 {
			t.Fatalf("PlayerList len = %d, want 3", len(info.PlayerList))
		}
		// Bot (ping 0)
		if info.Bots != 1 {
			t.Errorf("Bots = %d, want 1", info.Bots)
		}
	})

	t.Run("password protected", func(t *testing.T) {
		data := []byte("\xFF\xFF\xFF\xFFstatusResponse\n\\sv_hostname\\Private\\g_needpass\\1\\sv_maxclients\\8\n")

		info, err := parseQ3StatusResponse(data, 27960)
		if err != nil {
			t.Fatal(err)
		}
		if info.Visibility != "private" {
			t.Errorf("Visibility = %q, want %q", info.Visibility, "private")
		}
	})

	t.Run("raw color codes preserved", func(t *testing.T) {
		data := []byte("\xFF\xFF\xFF\xFFstatusResponse\n\\sv_hostname\\^1Red ^7Server\\mapname\\dm1\n")

		info, err := parseQ3StatusResponse(data, 27960)
		if err != nil {
			t.Fatal(err)
		}
		if info.Name != "^1Red ^7Server" {
			t.Errorf("Name = %q, want %q (raw color codes preserved)", info.Name, "^1Red ^7Server")
		}
	})
}
