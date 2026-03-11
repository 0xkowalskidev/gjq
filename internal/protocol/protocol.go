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

type ServerInfo struct {
	Protocol    string       `json:"protocol"`
	Name        string       `json:"name"`
	Map         string       `json:"map"`
	Game        string       `json:"game"`
	Players     int          `json:"players"`
	MaxPlayers  int          `json:"maxPlayers"`
	Bots        int          `json:"bots"`
	ServerType  string       `json:"serverType,omitempty"`
	Environment string       `json:"environment,omitempty"`
	Visibility  string       `json:"visibility,omitempty"`
	VAC         bool         `json:"vac,omitempty"`
	Version     string       `json:"version,omitempty"`
	Ping        Duration     `json:"ping"`
	Address     string       `json:"address"`
	Port        uint16       `json:"port"`
	GamePort    uint16       `json:"gamePort,omitempty"`
	PlayerList  []PlayerInfo `json:"playerList,omitempty"`
}

type PlayerInfo struct {
	Name     string   `json:"name"`
	Score    int      `json:"score"`
	Duration Duration `json:"duration"`
}

type QueryOpts struct {
	Players    bool
	ResolvedIP string // pre-resolved IP to skip redundant DNS lookups
}

type Querier interface {
	Query(ctx context.Context, address string, port uint16, opts QueryOpts) (*ServerInfo, error)
}

var registry = make(map[string]Querier)

func Register(name string, q Querier) {
	registry[name] = q
}

func Get(name string) (Querier, error) {
	q, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("protocol %q not registered", name)
	}
	return q, nil
}

func All() map[string]Querier {
	return registry
}
