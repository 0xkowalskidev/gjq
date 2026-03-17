package protocol

import (
	"encoding/binary"
	"testing"
	"time"
)

// buildPongPacket constructs a valid RakNet Unconnected Pong packet from a status string.
func buildPongPacket(status string) []byte {
	// 1 (id) + 8 (time) + 8 (guid) + 16 (magic) + 2 (strlen) + status
	pkt := make([]byte, 35+len(status))
	pkt[0] = 0x1C
	// bytes 1-8: time (zeros ok)
	// bytes 9-16: guid (zeros ok)
	copy(pkt[17:33], raknetMagic)
	binary.BigEndian.PutUint16(pkt[33:35], uint16(len(status)))
	copy(pkt[35:], status)
	return pkt
}

func TestParsePong(t *testing.T) {
	tests := []struct {
		name        string
		status      string
		wantName    string
		wantGame    string
		wantMap     string
		wantMode    string
		wantPlayers int
		wantMax     int
		wantVer     string
		wantErr     bool
	}{
		{
			name:        "full bedrock response",
			status:      "MCPE;My Server;486;1.19.50;3;20;12345;LevelName;Survival;1;19132;-1",
			wantName:    "My Server",
			wantGame:    "Minecraft: Bedrock Edition",
			wantMap:     "LevelName",
			wantMode:    "Survival",
			wantPlayers: 3,
			wantMax:     20,
			wantVer:     "1.19.50",
		},
		{
			name:        "education edition",
			status:      "MCEE;Edu Server;100;1.18.0;5;30;99999;World;Creative",
			wantName:    "Edu Server",
			wantGame:    "Minecraft: Education Edition",
			wantMap:     "World",
			wantMode:    "Creative",
			wantPlayers: 5,
			wantMax:     30,
			wantVer:     "1.18.0",
		},
		{
			name:        "minimal 6 fields",
			status:      "MCPE;Server;100;1.20.0;10;50",
			wantName:    "Server",
			wantGame:    "Minecraft: Bedrock Edition",
			wantPlayers: 10,
			wantMax:     50,
			wantVer:     "1.20.0",
		},
		{
			name:    "too few fields",
			status:  "MCPE;Server;100",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := buildPongPacket(tt.status)
			info, err := parsePong(pkt, "127.0.0.1", 19132, 10*time.Millisecond)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if info.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", info.Name, tt.wantName)
			}
			if info.Game != tt.wantGame {
				t.Errorf("Game = %q, want %q", info.Game, tt.wantGame)
			}
			if info.Map != tt.wantMap {
				t.Errorf("Map = %q, want %q", info.Map, tt.wantMap)
			}
			if info.GameMode != tt.wantMode {
				t.Errorf("GameMode = %q, want %q", info.GameMode, tt.wantMode)
			}
			if info.Players != tt.wantPlayers {
				t.Errorf("Players = %d, want %d", info.Players, tt.wantPlayers)
			}
			if info.MaxPlayers != tt.wantMax {
				t.Errorf("MaxPlayers = %d, want %d", info.MaxPlayers, tt.wantMax)
			}
			if info.Version != tt.wantVer {
				t.Errorf("Version = %q, want %q", info.Version, tt.wantVer)
			}
		})
	}
}

func TestParsePongExtraFields(t *testing.T) {
	// Full response with all 12 fields including port in field 10
	pkt := buildPongPacket("MCPE;My Server;486;1.19.50;3;20;12345;LevelName;Survival;1;25565;-1")
	info, err := parsePong(pkt, "127.0.0.1", 19132, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Extra["edition"] != "MCPE" {
		t.Errorf("Extra[edition] = %v, want %q", info.Extra["edition"], "MCPE")
	}
	if info.Extra["serverUniqueId"] != "12345" {
		t.Errorf("Extra[serverUniqueId] = %v, want %q", info.Extra["serverUniqueId"], "12345")
	}
	// Field 10 contains port 25565 — should override the default
	if info.GamePort != 25565 {
		t.Errorf("GamePort = %d, want 25565 (overridden by field 10)", info.GamePort)
	}
}

func TestParsePongMinimalNoExtras(t *testing.T) {
	// 6 fields only — no serverUniqueId, no map, no gameMode
	pkt := buildPongPacket("MCPE;Server;100;1.20.0;10;50")
	info, err := parsePong(pkt, "127.0.0.1", 19132, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := info.Extra["serverUniqueId"]; ok {
		t.Error("minimal response should not have serverUniqueId in Extra")
	}
	if info.Map != "" {
		t.Errorf("Map = %q, want empty for minimal response", info.Map)
	}
	if info.GameMode != "" {
		t.Errorf("GameMode = %q, want empty for minimal response", info.GameMode)
	}
}

func TestParsePongMalformed(t *testing.T) {
	t.Run("packet too short", func(t *testing.T) {
		_, err := parsePong([]byte{0x1C, 0x00}, "127.0.0.1", 19132, 0)
		if err == nil {
			t.Fatal("expected error for short packet")
		}
	})

	t.Run("wrong packet id", func(t *testing.T) {
		pkt := buildPongPacket("MCPE;S;1;1;0;0")
		pkt[0] = 0x01 // wrong id
		_, err := parsePong(pkt, "127.0.0.1", 19132, 0)
		if err == nil {
			t.Fatal("expected error for wrong packet id")
		}
	})
}
