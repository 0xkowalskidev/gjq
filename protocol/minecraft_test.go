package protocol

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
)

func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"plain string", "Hello World", "Hello World"},
		{"text object", map[string]interface{}{"text": "hello"}, "hello"},
		{
			"nested extra",
			map[string]interface{}{
				"text": "a",
				"extra": []interface{}{
					map[string]interface{}{"text": "b"},
					map[string]interface{}{"text": "c"},
				},
			},
			"abc",
		},
		{
			"deep nesting",
			map[string]interface{}{
				"text": "root",
				"extra": []interface{}{
					map[string]interface{}{
						"text": " > child",
						"extra": []interface{}{
							map[string]interface{}{"text": " > grandchild"},
						},
					},
				},
			},
			"root > child > grandchild",
		},
		{"nil input", nil, ""},
		{"empty string", "", ""},
		{"empty text object", map[string]interface{}{"text": ""}, ""},
		{
			"text with color codes",
			map[string]interface{}{"text": "§aGreen §bAqua"},
			"§aGreen §bAqua",
		},
		{"number input", 42, ""},
		{"bool input", true, ""},
		{
			"extra without text",
			map[string]interface{}{
				"extra": []interface{}{
					map[string]interface{}{"text": "child only"},
				},
			},
			"child only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDescription(tt.input)
			if got != tt.want {
				t.Errorf("extractDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadWriteVarInt(t *testing.T) {
	tests := []struct {
		name  string
		value int32
	}{
		{"zero", 0},
		{"one", 1},
		{"max single byte", 127},
		{"two bytes", 128},
		{"255", 255},
		{"minecraft port", 25565},
		{"negative one", -1},
		{"max int32", math.MaxInt32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writeVarInt(&buf, tt.value)

			got, err := readVarInt(&buf)
			if err != nil {
				t.Fatalf("readVarInt error: %v", err)
			}
			if got != tt.value {
				t.Errorf("round-trip: wrote %d, read %d", tt.value, got)
			}
		})
	}
}

func TestReadVarIntErrors(t *testing.T) {
	t.Run("empty buffer", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		_, err := readVarInt(buf)
		if err == nil {
			t.Fatal("expected error for empty buffer")
		}
	})

	t.Run("overlong varint", func(t *testing.T) {
		// 6 continuation bytes — exceeds 5-byte limit
		buf := bytes.NewBuffer([]byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x00})
		_, err := readVarInt(buf)
		if err == nil {
			t.Fatal("expected error for overlong varint")
		}
	})
}

func TestMCStatusResponseUnmarshal(t *testing.T) {
	jsonStr := `{
		"description": {"text": "A ", "extra": [{"text": "Minecraft Server"}]},
		"players": {"max": 20, "online": 5, "sample": [{"name": "Steve", "id": "abc-123"}, {"name": "Alex", "id": "def-456"}]},
		"version": {"name": "1.20.4", "protocol": 765},
		"favicon": "data:image/png;base64,abc",
		"enforcesSecureChat": true
	}`

	var status mcStatusResponse
	if err := json.Unmarshal([]byte(jsonStr), &status); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if status.Players.Online != 5 {
		t.Errorf("Players.Online = %d, want 5", status.Players.Online)
	}
	if status.Players.Max != 20 {
		t.Errorf("Players.Max = %d, want 20", status.Players.Max)
	}
	if len(status.Players.Sample) != 2 {
		t.Fatalf("Players.Sample len = %d, want 2", len(status.Players.Sample))
	}
	if status.Players.Sample[0].Name != "Steve" {
		t.Errorf("Sample[0].Name = %q, want %q", status.Players.Sample[0].Name, "Steve")
	}
	if status.Version.Name != "1.20.4" {
		t.Errorf("Version.Name = %q, want %q", status.Version.Name, "1.20.4")
	}
	if status.Favicon != "data:image/png;base64,abc" {
		t.Errorf("Favicon = %q, want non-empty", status.Favicon)
	}
	if !status.EnforcesSecureChat {
		t.Error("EnforcesSecureChat = false, want true")
	}

	// Verify description extraction works with the parsed description
	desc := extractDescription(status.Description)
	if desc != "A Minecraft Server" {
		t.Errorf("extractDescription = %q, want %q", desc, "A Minecraft Server")
	}
}

func TestMCStatusResponseForgeMods(t *testing.T) {
	t.Run("forgeData FML2+", func(t *testing.T) {
		jsonStr := `{
			"description": "test",
			"players": {"max": 20, "online": 0},
			"version": {"name": "1.20.1", "protocol": 763},
			"forgeData": {"mods": [{"modId": "forge", "modmarker": "47.2.0"}, {"modId": "mymod", "modmarker": "1.0.0"}]}
		}`
		var status mcStatusResponse
		if err := json.Unmarshal([]byte(jsonStr), &status); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if status.ForgeData == nil {
			t.Fatal("ForgeData is nil")
		}
		if len(status.ForgeData.Mods) != 2 {
			t.Fatalf("ForgeData.Mods len = %d, want 2", len(status.ForgeData.Mods))
		}
		if status.ForgeData.Mods[0].ModID != "forge" {
			t.Errorf("Mods[0].ModID = %q, want %q", status.ForgeData.Mods[0].ModID, "forge")
		}
		if status.ForgeData.Mods[1].Version != "1.0.0" {
			t.Errorf("Mods[1].Version = %q, want %q", status.ForgeData.Mods[1].Version, "1.0.0")
		}
	})

	t.Run("modinfo FML legacy", func(t *testing.T) {
		jsonStr := `{
			"description": "test",
			"players": {"max": 20, "online": 0},
			"version": {"name": "1.12.2", "protocol": 340},
			"modinfo": {"modList": [{"modid": "forge", "version": "14.23.5.2860"}, {"modid": "oldmod", "version": "2.0"}]}
		}`
		var status mcStatusResponse
		if err := json.Unmarshal([]byte(jsonStr), &status); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if status.ModInfo == nil {
			t.Fatal("ModInfo is nil")
		}
		if len(status.ModInfo.ModList) != 2 {
			t.Fatalf("ModInfo.ModList len = %d, want 2", len(status.ModInfo.ModList))
		}
		if status.ModInfo.ModList[0].ModID != "forge" {
			t.Errorf("ModList[0].ModID = %q, want %q", status.ModInfo.ModList[0].ModID, "forge")
		}
	})

	t.Run("no mods", func(t *testing.T) {
		jsonStr := `{
			"description": "vanilla",
			"players": {"max": 20, "online": 0},
			"version": {"name": "1.20.4", "protocol": 765}
		}`
		var status mcStatusResponse
		if err := json.Unmarshal([]byte(jsonStr), &status); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if status.ForgeData != nil {
			t.Error("ForgeData should be nil for vanilla server")
		}
		if status.ModInfo != nil {
			t.Error("ModInfo should be nil for vanilla server")
		}
	})
}
