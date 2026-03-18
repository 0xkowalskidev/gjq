package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

type ModInfo struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type ServerInfo struct {
	Protocol    string       `json:"protocol"`
	Name        string       `json:"name"`
	Map         string       `json:"map"`
	Game        string       `json:"game"`
	GameMode    string       `json:"gameMode,omitempty"`
	Players     int          `json:"players"`
	MaxPlayers  int          `json:"maxPlayers"`
	Bots        int          `json:"bots"`
	ServerType  string       `json:"serverType,omitempty"`
	Environment string       `json:"environment,omitempty"`
	Visibility  string       `json:"visibility,omitempty"`
	VAC         bool         `json:"vac,omitempty"`
	Version     string       `json:"version,omitempty"`
	Keywords    string       `json:"keywords,omitempty"`
	AppID       uint32       `json:"appId,omitempty"`
	Ping        Duration     `json:"ping"`
	Address     string       `json:"address"`
	GamePort         uint16       `json:"gamePort"`
	ReportedGamePort uint16       `json:"reportedGamePort,omitempty"`
	QueryPort        uint16       `json:"queryPort"`
	PlayerList  []PlayerInfo `json:"playerList,omitempty"`
	Rules       map[string]string `json:"rules,omitempty"`
	Mods        []ModInfo         `json:"mods,omitempty"`
	Extra       map[string]any    `json:"extra,omitempty"`
}

type PlayerInfo struct {
	Name     string   `json:"name"`
	Score    int      `json:"score"`
	Duration Duration `json:"duration"`
}

type EOSConfig struct {
	ClientID        string
	ClientSecret    string
	DeploymentID    string
	UseExternalAuth bool              // false=client_credentials, true=device ID flow
	UseWildcard     bool              // use /wildcard/matchmaking/ endpoint
	Attributes      map[string]string // canonical name -> game-specific attribute key
}

type QueryOpts struct {
	Players    bool
	Rules      bool
	ResolvedIP string // pre-resolved IP to skip redundant DNS lookups
	EOS        *EOSConfig
}

type Querier interface {
	Query(ctx context.Context, address string, port uint16, opts QueryOpts) (*ServerInfo, error)
}

// Truncate returns s truncated to n characters with "..." appended if needed.
func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

var queriers = map[string]Querier{}

func Register(name string, q Querier) {
	queriers[name] = q
}

func Get(name string) (Querier, error) {
	q, ok := queriers[name]
	if !ok {
		return nil, fmt.Errorf("protocol %q not registered", name)
	}
	return q, nil
}

func All() map[string]Querier {
	cp := make(map[string]Querier, len(queriers))
	for k, v := range queriers {
		cp[k] = v
	}
	return cp
}
