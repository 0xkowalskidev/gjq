package tshock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/0xkowalskidev/gsq/internal/protocol"
)

type tshockQuerier struct{}

func init() {
	protocol.Register("tshock", &tshockQuerier{})
}

type statusResponse struct {
	Name           string `json:"name"`
	Port           int    `json:"port"`
	PlayerCount    int    `json:"playercount"`
	MaxPlayers     int    `json:"maxplayers"`
	World          string `json:"world"`
	Players        string `json:"players"`
	ServerVersion  string `json:"serverversion"`
	ServerPassword bool   `json:"serverpassword"`
	TShockVersion  string `json:"tshockversion"`
	Uptime         string `json:"uptime"`
}

func (q *tshockQuerier) Query(ctx context.Context, address string, port uint16, opts protocol.QueryOpts) (*protocol.ServerInfo, error) {
	host := address
	if opts.ResolvedIP != "" {
		host = opts.ResolvedIP
	}

	endpoint := fmt.Sprintf("http://%s:%d/status", host, port)
	slog.Debug("tshock: querying server", "endpoint", endpoint)

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tshock query %s:%d: %w", address, port, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tshock read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tshock HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var status statusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("tshock parse response: %w", err)
	}

	ping := time.Since(start)

	name := status.Name
	if name == "" {
		name = status.World
	}

	visibility := "public"
	if status.ServerPassword {
		visibility = "private"
	}

	info := &protocol.ServerInfo{
		Protocol:   "tshock",
		Name:       name,
		Map:        status.World,
		Players:    status.PlayerCount,
		MaxPlayers: status.MaxPlayers,
		Version:    status.ServerVersion,
		Visibility: visibility,
		GamePort:   uint16(status.Port),
		QueryPort:  port,
		Ping:       protocol.Duration{Duration: ping},
	}

	if status.TShockVersion != "" || status.Uptime != "" {
		info.Extra = make(map[string]any)
		if status.TShockVersion != "" {
			info.Extra["tshockVersion"] = status.TShockVersion
		}
		if status.Uptime != "" {
			info.Extra["uptime"] = status.Uptime
		}
	}

	if opts.Players && status.Players != "" {
		for _, name := range splitPlayers(status.Players) {
			if name != "" {
				info.PlayerList = append(info.PlayerList, protocol.PlayerInfo{Name: name})
			}
		}
	}

	return info, nil
}

// splitPlayers splits the comma-separated player string from the /status endpoint.
func splitPlayers(s string) []string {
	var players []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			p := trim(s[start:i])
			if p != "" {
				players = append(players, p)
			}
			start = i + 1
		}
	}
	p := trim(s[start:])
	if p != "" {
		players = append(players, p)
	}
	return players
}

func trim(s string) string {
	start, end := 0, len(s)
	for start < end && s[start] == ' ' {
		start++
	}
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
