package protocol

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type quake3Querier struct{}

func init() {
	Register("quake3", &quake3Querier{})
}

func (q *quake3Querier) Query(ctx context.Context, address string, port uint16, opts QueryOpts) (*ServerInfo, error) {
	host := address
	if opts.ResolvedIP != "" {
		host = opts.ResolvedIP
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}

	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP(host), Port: int(port)})
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(deadline)

	start := time.Now()

	info, err := q3QueryStatus(conn, port)
	if err != nil {
		return nil, fmt.Errorf("quake3 query %s:%d: %w", address, port, err)
	}

	info.Ping = Duration{Duration: time.Since(start)}
	return info, nil
}

func q3QueryStatus(conn *net.UDPConn, port uint16) (*ServerInfo, error) {
	packet := []byte("\xFF\xFF\xFF\xFFgetstatus\x00")
	if _, err := conn.Write(packet); err != nil {
		return nil, fmt.Errorf("send getstatus: %w", err)
	}

	buf := make([]byte, 8192)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return parseQ3StatusResponse(buf[:n], port)
}

func parseQ3StatusResponse(data []byte, port uint16) (*ServerInfo, error) {
	prefix := "\xFF\xFF\xFF\xFFstatusResponse\n"
	if len(data) < len(prefix) || string(data[:len(prefix)]) != prefix {
		return nil, fmt.Errorf("invalid statusResponse header")
	}

	body := string(data[len(prefix):])
	parts := strings.SplitN(body, "\n", 2)

	vars := parseQ3InfoString(parts[0])
	info := mapQ3Vars(vars, port)

	if len(parts) > 1 {
		info.PlayerList = parseQ3Players(parts[1])

		// Count bots (ping == 0)
		bots := 0
		for _, p := range info.PlayerList {
			if p.Duration.Duration == 0 {
				bots++
			}
		}
		info.Bots = bots
	}

	return info, nil
}


func parseQ3InfoString(s string) map[string]string {
	vars := make(map[string]string)
	parts := strings.Split(s, "\\")

	// Skip leading empty element from leading backslash
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	for i := 0; i+1 < len(parts); i += 2 {
		vars[parts[i]] = parts[i+1]
	}
	return vars
}

func parseQ3Players(s string) []PlayerInfo {
	var players []PlayerInfo
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: score ping "name"
		quoteStart := strings.IndexByte(line, '"')
		if quoteStart < 0 {
			continue
		}
		quoteEnd := strings.LastIndexByte(line, '"')
		if quoteEnd <= quoteStart {
			continue
		}

		name := line[quoteStart+1 : quoteEnd]

		fields := strings.Fields(line[:quoteStart])
		if len(fields) < 2 {
			continue
		}

		score, _ := strconv.Atoi(fields[0])
		ping, _ := strconv.Atoi(fields[1])

		players = append(players, PlayerInfo{
			Name:     name,
			Score:    score,
			Duration: Duration{Duration: time.Duration(ping) * time.Millisecond},
		})
	}
	return players
}

func mapQ3Vars(vars map[string]string, port uint16) *ServerInfo {
	players, _ := strconv.Atoi(vars["clients"])
	maxPlayers, _ := strconv.Atoi(vars["sv_maxclients"])

	info := &ServerInfo{
		Protocol:   "quake3",
		Name:       vars["sv_hostname"],
		Map:        vars["mapname"],
		GameMode:   vars["gametype"],
		Players:    players,
		MaxPlayers: maxPlayers,
		GamePort:   port,
		QueryPort:  port,
		Version:    vars["shortversion"],
	}

	if v, ok := vars["g_needpass"]; ok && v == "1" {
		info.Visibility = "private"
	} else {
		info.Visibility = "public"
	}

	if v, ok := vars["game"]; ok && v != "" {
		info.Extra = map[string]any{"mod": v}
	}

	return info
}

