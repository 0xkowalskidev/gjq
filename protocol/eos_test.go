package protocol

import (
	"encoding/json"
	"testing"
)

func TestSessionAttributes(t *testing.T) {
	tests := []struct {
		name    string
		session map[string]json.RawMessage
		wantNil bool
		wantKey string
		wantVal string
	}{
		{
			name: "valid attributes",
			session: map[string]json.RawMessage{
				"attributes": json.RawMessage(`{"CUSTOMSERVERNAME_s":"My Server","MAPNAME_s":"TheIsland"}`),
			},
			wantKey: "CUSTOMSERVERNAME_s",
			wantVal: "My Server",
		},
		{
			name:    "missing attributes key",
			session: map[string]json.RawMessage{},
			wantNil: true,
		},
		{
			name: "malformed attributes JSON",
			session: map[string]json.RawMessage{
				"attributes": json.RawMessage(`not valid json`),
			},
			wantNil: true,
		},
		{
			name: "empty attributes object",
			session: map[string]json.RawMessage{
				"attributes": json.RawMessage(`{}`),
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sessionAttributes(tt.session)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("sessionAttributes() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("sessionAttributes() = nil, want non-nil")
			}
			if tt.wantKey != "" {
				if got[tt.wantKey] != tt.wantVal {
					t.Errorf("sessionAttributes()[%q] = %q, want %q", tt.wantKey, got[tt.wantKey], tt.wantVal)
				}
			}
		})
	}
}

func TestSessionAttributesTypes(t *testing.T) {
	session := map[string]json.RawMessage{
		"attributes": json.RawMessage(`{"str_val":"hello","num_val":42,"bool_val":true,"obj_val":{"nested":1}}`),
	}

	attrs := sessionAttributes(session)
	if attrs == nil {
		t.Fatal("sessionAttributes() = nil")
	}

	if attrs["str_val"] != "hello" {
		t.Errorf("str_val = %q, want %q", attrs["str_val"], "hello")
	}
	if attrs["num_val"] != "42" {
		t.Errorf("num_val = %q, want %q", attrs["num_val"], "42")
	}
	if attrs["bool_val"] != "true" {
		t.Errorf("bool_val = %q, want %q", attrs["bool_val"], "true")
	}
	// Nested object falls back to raw JSON string
	if attrs["obj_val"] != `{"nested":1}` {
		t.Errorf("obj_val = %q, want raw JSON", attrs["obj_val"])
	}
}

func TestMapSession(t *testing.T) {
	attrMap := map[string]string{
		"name":     "CUSTOMSERVERNAME_s",
		"map":      "MAPNAME_s",
		"password": "SERVERPASSWORD_b",
		"version":  "BUILDID_s",
	}

	t.Run("full session", func(t *testing.T) {
		session := map[string]json.RawMessage{
			"attributes":    json.RawMessage(`{"CUSTOMSERVERNAME_s":"ARK Server","MAPNAME_s":"TheIsland","SERVERPASSWORD_b":"false","BUILDID_s":"12345"}`),
			"totalPlayers":  json.RawMessage(`10`),
			"publicPlayers": json.RawMessage(`["EOS_001","EOS_002"]`),
			"settings":      json.RawMessage(`{"maxPublicPlayers":70}`),
		}

		info := mapSession(session, attrMap)
		if info.Name != "ARK Server" {
			t.Errorf("Name = %q, want %q", info.Name, "ARK Server")
		}
		if info.Map != "TheIsland" {
			t.Errorf("Map = %q, want %q", info.Map, "TheIsland")
		}
		if info.Version != "12345" {
			t.Errorf("Version = %q, want %q", info.Version, "12345")
		}
		if info.Visibility != "public" {
			t.Errorf("Visibility = %q, want %q", info.Visibility, "public")
		}
		if info.Players != 10 {
			t.Errorf("Players = %d, want 10", info.Players)
		}
		if info.MaxPlayers != 70 {
			t.Errorf("MaxPlayers = %d, want 70", info.MaxPlayers)
		}
		if len(info.PlayerList) != 2 {
			t.Fatalf("PlayerList len = %d, want 2", len(info.PlayerList))
		}
		if info.PlayerList[0].Name != "EOS_001" {
			t.Errorf("PlayerList[0].Name = %q, want %q", info.PlayerList[0].Name, "EOS_001")
		}
	})

	t.Run("password protected", func(t *testing.T) {
		session := map[string]json.RawMessage{
			"attributes": json.RawMessage(`{"CUSTOMSERVERNAME_s":"Private","SERVERPASSWORD_b":"true"}`),
		}
		info := mapSession(session, attrMap)
		if info.Visibility != "private" {
			t.Errorf("Visibility = %q, want %q", info.Visibility, "private")
		}
	})

	t.Run("missing attributes", func(t *testing.T) {
		session := map[string]json.RawMessage{}
		info := mapSession(session, attrMap)
		// Should not panic — just return empty ServerInfo
		if info == nil {
			t.Fatal("mapSession returned nil")
		}
	})
}

func TestJsonInt(t *testing.T) {
	session := map[string]json.RawMessage{
		"totalPlayers": json.RawMessage(`10`),
		"settings":     json.RawMessage(`{"maxPublicPlayers":70}`),
		"strVal":       json.RawMessage(`"not a number"`),
	}

	if got := jsonInt(session, "totalPlayers"); got != 10 {
		t.Errorf("jsonInt(totalPlayers) = %d, want 10", got)
	}
	if got := jsonInt(session, "settings", "maxPublicPlayers"); got != 70 {
		t.Errorf("jsonInt(settings.maxPublicPlayers) = %d, want 70", got)
	}
	if got := jsonInt(session, "missing"); got != 0 {
		t.Errorf("jsonInt(missing) = %d, want 0", got)
	}
	if got := jsonInt(session, "strVal"); got != 0 {
		t.Errorf("jsonInt(strVal) = %d, want 0", got)
	}
	if got := jsonInt(session); got != 0 {
		t.Errorf("jsonInt(no keys) = %d, want 0", got)
	}
}
