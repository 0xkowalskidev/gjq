package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type tshockQuerier struct{}

func init() {
	Register("tshock", &tshockQuerier{})
}

type tshockStatusResponse struct {
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

func (q *tshockQuerier) Query(ctx context.Context, address string, port uint16, opts QueryOpts) (*ServerInfo, error) {
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
		return nil, fmt.Errorf("tshock HTTP %d: %s", resp.StatusCode, Truncate(string(body), 200))
	}

	var status tshockStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("tshock parse response: %w", err)
	}

	ping := time.Since(start)
	return mapTshockStatus(status, port, ping, opts.Players), nil
}

func mapTshockStatus(status tshockStatusResponse, port uint16, ping time.Duration, players bool) *ServerInfo {
	name := status.Name
	if name == "" {
		name = status.World
	}

	visibility := "public"
	if status.ServerPassword {
		visibility = "private"
	}

	info := &ServerInfo{
		Protocol:   "tshock",
		Name:       name,
		Map:        status.World,
		Players:    status.PlayerCount,
		MaxPlayers: status.MaxPlayers,
		Version:    status.ServerVersion,
		Visibility: visibility,
		GamePort:         uint16(status.Port),
		ReportedGamePort: uint16(status.Port),
		QueryPort:        port,
		Ping:       Duration{Duration: ping},
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

	if players && status.Players != "" {
		for _, name := range splitPlayers(status.Players) {
			if name != "" {
				info.PlayerList = append(info.PlayerList, PlayerInfo{Name: name})
			}
		}
	}

	return info
}

// splitPlayers splits the comma-separated player string from the /status endpoint.
func splitPlayers(s string) []string {
	var players []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			p := strings.TrimSpace(s[start:i])
			if p != "" {
				players = append(players, p)
			}
			start = i + 1
		}
	}
	p := strings.TrimSpace(s[start:])
	if p != "" {
		players = append(players, p)
	}
	return players
}
