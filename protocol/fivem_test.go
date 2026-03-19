package protocol

import (
	"testing"
	"time"
)

func TestMapFivemResponse(t *testing.T) {
	t.Run("full response", func(t *testing.T) {
		dynamic := &fivemDynamic{
			Hostname:   "My FiveM Server",
			GameType:   "Roleplay",
			MapName:    "San Andreas",
			Clients:    32,
			MaxClients: "48",
		}
		serverInfo := &fivemInfo{
			Server: "FXServer-master SERVER v1.0.0.17000 win32",
			Vars: map[string]string{
				"tags":              "roleplay,serious",
				"locale":            "en-US",
				"sv_projectName":    "My RP",
				"sv_projectDesc":    "A cool server",
				"onesync_enabled":   "true",
			},
		}
		players := []fivemPlayer{
			{Name: "Alice", Ping: 30, ID: 1},
			{Name: "Bob", Ping: 45, ID: 2},
		}

		info := mapFivemResponse(dynamic, serverInfo, players, 30120, 50*time.Millisecond, true)

		if info.Protocol != "fivem" {
			t.Errorf("Protocol = %q, want %q", info.Protocol, "fivem")
		}
		if info.Name != "My FiveM Server" {
			t.Errorf("Name = %q, want %q", info.Name, "My FiveM Server")
		}
		if info.Map != "San Andreas" {
			t.Errorf("Map = %q, want %q", info.Map, "San Andreas")
		}
		if info.GameMode != "Roleplay" {
			t.Errorf("GameMode = %q, want %q", info.GameMode, "Roleplay")
		}
		if info.Players != 32 {
			t.Errorf("Players = %d, want 32", info.Players)
		}
		if info.MaxPlayers != 48 {
			t.Errorf("MaxPlayers = %d, want 48", info.MaxPlayers)
		}
		if info.Version != "FXServer-master SERVER v1.0.0.17000 win32" {
			t.Errorf("Version = %q, want FXServer string", info.Version)
		}
		if info.Keywords != "roleplay,serious" {
			t.Errorf("Keywords = %q, want %q", info.Keywords, "roleplay,serious")
		}
		if info.GamePort != 30120 {
			t.Errorf("GamePort = %d, want 30120", info.GamePort)
		}
		if info.QueryPort != 30120 {
			t.Errorf("QueryPort = %d, want 30120", info.QueryPort)
		}
		if info.Extra["locale"] != "en-US" {
			t.Errorf("Extra[locale] = %v, want %q", info.Extra["locale"], "en-US")
		}
		if info.Extra["projectName"] != "My RP" {
			t.Errorf("Extra[projectName] = %v, want %q", info.Extra["projectName"], "My RP")
		}
		if info.Extra["onesync"] != "true" {
			t.Errorf("Extra[onesync] = %v, want %q", info.Extra["onesync"], "true")
		}
		if len(info.PlayerList) != 2 {
			t.Fatalf("PlayerList len = %d, want 2", len(info.PlayerList))
		}
		if info.PlayerList[0].Name != "Alice" {
			t.Errorf("PlayerList[0].Name = %q, want %q", info.PlayerList[0].Name, "Alice")
		}
		if info.PlayerList[1].Name != "Bob" {
			t.Errorf("PlayerList[1].Name = %q, want %q", info.PlayerList[1].Name, "Bob")
		}
	})

	t.Run("nil server info", func(t *testing.T) {
		dynamic := &fivemDynamic{
			Hostname:   "Basic Server",
			Clients:    5,
			MaxClients: "32",
		}

		info := mapFivemResponse(dynamic, nil, nil, 30120, 10*time.Millisecond, false)

		if info.Name != "Basic Server" {
			t.Errorf("Name = %q, want %q", info.Name, "Basic Server")
		}
		if info.Version != "" {
			t.Errorf("Version = %q, want empty", info.Version)
		}
		if info.Extra != nil {
			t.Errorf("Extra = %v, want nil", info.Extra)
		}
	})

	t.Run("players not fetched when disabled", func(t *testing.T) {
		dynamic := &fivemDynamic{Hostname: "S", MaxClients: "10"}
		players := []fivemPlayer{{Name: "Alice"}}

		info := mapFivemResponse(dynamic, nil, players, 30120, 0, false)

		if len(info.PlayerList) != 0 {
			t.Errorf("PlayerList len = %d, want 0 (players disabled)", len(info.PlayerList))
		}
	})

	t.Run("invalid max clients", func(t *testing.T) {
		dynamic := &fivemDynamic{Hostname: "S", MaxClients: "not_a_number"}

		info := mapFivemResponse(dynamic, nil, nil, 30120, 0, false)

		if info.MaxPlayers != 0 {
			t.Errorf("MaxPlayers = %d, want 0 for invalid input", info.MaxPlayers)
		}
	})

	t.Run("no extra when vars empty", func(t *testing.T) {
		dynamic := &fivemDynamic{Hostname: "S", MaxClients: "10"}
		serverInfo := &fivemInfo{
			Server: "FXServer",
			Vars:   map[string]string{},
		}

		info := mapFivemResponse(dynamic, serverInfo, nil, 30120, 0, false)

		if info.Extra != nil {
			t.Errorf("Extra = %v, want nil when vars has no relevant keys", info.Extra)
		}
	})
}
