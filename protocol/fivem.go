package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type fivemQuerier struct{}

func init() {
	Register("fivem", &fivemQuerier{})
}

type fivemDynamic struct {
	Hostname     string `json:"hostname"`
	GameType     string `json:"gametype"`
	MapName      string `json:"mapname"`
	Clients      int    `json:"clients"`
	MaxClients   string `json:"sv_maxclients"`
}

type fivemPlayer struct {
	Name string `json:"name"`
	Ping int    `json:"ping"`
	ID   int    `json:"id"`
}

type fivemInfo struct {
	Server string            `json:"server"`
	Vars   map[string]string `json:"vars"`
}

func (q *fivemQuerier) Query(ctx context.Context, address string, port uint16, opts QueryOpts) (*ServerInfo, error) {
	host := address
	if opts.ResolvedIP != "" {
		host = opts.ResolvedIP
	}

	base := fmt.Sprintf("http://%s:%d", host, port)

	start := time.Now()

	dynamic, err := fivemGet[fivemDynamic](ctx, base+"/dynamic.json")
	if err != nil {
		return nil, fmt.Errorf("fivem query %s:%d: %w", address, port, err)
	}

	ping := time.Since(start)

	// Fetch server metadata (optional, non-fatal)
	serverInfo, err := fivemGet[fivemInfo](ctx, base+"/info.json")
	if err != nil {
		slog.Debug("fivem: info.json failed, continuing without metadata", "error", err)
	}

	// Fetch player list (optional, may be auth-protected)
	var players []fivemPlayer
	if opts.Players {
		result, err := fivemGet[[]fivemPlayer](ctx, base+"/players.json")
		if err != nil {
			slog.Debug("fivem: players.json failed, may be auth-protected", "error", err)
		} else {
			players = *result
		}
	}

	return mapFivemResponse(dynamic, serverInfo, players, port, ping, opts.Players), nil
}

func mapFivemResponse(dynamic *fivemDynamic, serverInfo *fivemInfo, players []fivemPlayer, port uint16, ping time.Duration, fetchPlayers bool) *ServerInfo {
	maxPlayers, _ := strconv.Atoi(dynamic.MaxClients)

	info := &ServerInfo{
		Protocol:   "fivem",
		Name:       dynamic.Hostname,
		Map:        dynamic.MapName,
		GameMode:   dynamic.GameType,
		Players:    dynamic.Clients,
		MaxPlayers: maxPlayers,
		GamePort:   port,
		QueryPort:  port,
		Ping:       Duration{Duration: ping},
	}

	if serverInfo != nil {
		info.Version = serverInfo.Server
		extra := make(map[string]any)
		if v, ok := serverInfo.Vars["tags"]; ok {
			info.Keywords = v
		}
		if v, ok := serverInfo.Vars["locale"]; ok {
			extra["locale"] = v
		}
		if v, ok := serverInfo.Vars["sv_projectName"]; ok {
			extra["projectName"] = v
		}
		if v, ok := serverInfo.Vars["sv_projectDesc"]; ok {
			extra["projectDesc"] = v
		}
		if v, ok := serverInfo.Vars["onesync_enabled"]; ok {
			extra["onesync"] = v
		}
		if len(extra) > 0 {
			info.Extra = extra
		}
	}

	if fetchPlayers {
		for _, p := range players {
			info.PlayerList = append(info.PlayerList, PlayerInfo{Name: p.Name})
		}
	}

	return info
}

func fivemGet[T any](ctx context.Context, url string) (*T, error) {
	slog.Debug("fivem: fetching", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, Truncate(string(body), 200))
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &result, nil
}
