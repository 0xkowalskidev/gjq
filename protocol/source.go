package protocol

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	a2sInfoRequest  = 0x54
	a2sInfoResponse = 0x49

	a2sPlayerRequest  = 0x55
	a2sPlayerResponse = 0x44

	a2sRulesRequest  = 0x56
	a2sRulesResponse = 0x45

	challengeResponse = 0x41

	singlePacketHeader = 0xFFFFFFFF
	splitPacketHeader  = 0xFFFFFFFE
)

var a2sInfoPayload = append(
	[]byte{0xFF, 0xFF, 0xFF, 0xFF, a2sInfoRequest},
	append([]byte("Source Engine Query"), 0x00)...,
)

type SourceQuerier struct{}

func init() {
	Register("source", &SourceQuerier{})
}

func (q *SourceQuerier) Query(ctx context.Context, address string, port uint16, opts QueryOpts) (*ServerInfo, error) {
	dialHost := address
	if opts.ResolvedIP != "" {
		dialHost = opts.ResolvedIP
	}
	addr := net.JoinHostPort(dialHost, fmt.Sprintf("%d", port))

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}

	conn, err := net.DialTimeout("udp", addr, time.Until(deadline))
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer conn.Close()

	conn.SetDeadline(deadline)

	info, ping, err := queryInfo(conn)
	if err != nil {
		return nil, fmt.Errorf("a2s_info %s: %w", addr, err)
	}

	info.Ping = Duration{Duration: ping}
	info.GamePort = port
	info.QueryPort = port

	if opts.Players {
		// Use remaining context deadline for player query. Some servers don't
		// respond to A2S_PLAYER at all, so the error is swallowed below.
		conn.SetDeadline(deadline)

		players, err := queryPlayers(conn)
		if err != nil {
			slog.Debug("a2s_player query failed", "error", err)
		} else {
			info.PlayerList = players
		}
	}

	if opts.Rules {
		conn.SetDeadline(deadline)

		rules, err := queryRules(conn)
		if err != nil {
			slog.Debug("a2s_rules query failed", "error", err)
		} else {
			info.Rules = rules
		}
	}

	return info, nil
}

// readResponse reads a single or multi-packet response from the connection.
// Multi-packet responses (header 0xFEFFFFFF) are reassembled from fragments.
func readResponse(conn net.Conn) ([]byte, error) {
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if n < 4 {
		return nil, fmt.Errorf("response too short: %d bytes", n)
	}

	header := binary.LittleEndian.Uint32(buf[:4])
	if header == singlePacketHeader {
		return buf[4:n], nil
	}
	if header != splitPacketHeader {
		return nil, fmt.Errorf("unexpected header: 0x%08x", header)
	}

	// Multi-packet response
	return readSplitResponse(conn, buf[:n])
}

func readSplitResponse(conn net.Conn, firstPacket []byte) ([]byte, error) {
	type fragment struct {
		number  byte
		payload []byte
	}

	parseFragment := func(pkt []byte) (requestID int32, total byte, frag fragment, err error) {
		if len(pkt) < 12 {
			return 0, 0, fragment{}, fmt.Errorf("split packet too short: %d bytes", len(pkt))
		}
		r := bytes.NewReader(pkt[4:]) // skip 0xFFFFFFFE header
		binary.Read(r, binary.LittleEndian, &requestID)
		total, _ = r.ReadByte()
		number, _ := r.ReadByte()
		r.Seek(2, io.SeekCurrent) // skip max packet size (Source engine split format)
		payload := make([]byte, r.Len())
		r.Read(payload)
		return requestID, total, fragment{number: number, payload: payload}, nil
	}

	reqID, total, first, err := parseFragment(firstPacket)
	if err != nil {
		return nil, err
	}

	slog.Debug("a2s: split response", "requestID", reqID, "total", total, "received", first.number)

	fragments := make([]fragment, 0, total)
	fragments = append(fragments, first)

	buf := make([]byte, 4096)
	for len(fragments) < int(total) {
		n, err := conn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("read split packet %d/%d: %w", len(fragments)+1, total, err)
		}

		_, _, frag, err := parseFragment(buf[:n])
		if err != nil {
			return nil, err
		}
		fragments = append(fragments, frag)
	}

	// Reassemble in order
	sort.Slice(fragments, func(i, j int) bool {
		return fragments[i].number < fragments[j].number
	})

	var assembled bytes.Buffer
	for _, f := range fragments {
		assembled.Write(f.payload)
	}

	// Assembled data starts with the same format as single-packet (0xFFFFFFFF + type byte)
	data := assembled.Bytes()
	if len(data) < 5 {
		return nil, fmt.Errorf("reassembled response too short: %d bytes", len(data))
	}
	// Skip the inner 0xFFFFFFFF header
	if binary.LittleEndian.Uint32(data[:4]) == singlePacketHeader {
		return data[4:], nil
	}
	return data, nil
}

func queryInfo(conn net.Conn) (*ServerInfo, time.Duration, error) {
	start := time.Now()
	if _, err := conn.Write(a2sInfoPayload); err != nil {
		return nil, 0, fmt.Errorf("send request: %w", err)
	}

	data, err := readResponse(conn)
	if err != nil {
		return nil, 0, fmt.Errorf("read response: %w", err)
	}
	ping := time.Since(start)

	if len(data) < 1 {
		return nil, 0, fmt.Errorf("response too short")
	}

	if data[0] == challengeResponse {
		if len(data) < 5 {
			return nil, 0, fmt.Errorf("challenge response too short: %d bytes", len(data))
		}
		challenge := data[1:5]
		retryPayload := make([]byte, len(a2sInfoPayload), len(a2sInfoPayload)+4)
		copy(retryPayload, a2sInfoPayload)
		retryPayload = append(retryPayload, challenge...)

		start = time.Now()
		if _, err := conn.Write(retryPayload); err != nil {
			return nil, 0, fmt.Errorf("send challenge response: %w", err)
		}

		data, err = readResponse(conn)
		if err != nil {
			return nil, 0, fmt.Errorf("read challenge reply: %w", err)
		}
		ping = time.Since(start)
	}

	if len(data) < 1 || data[0] != a2sInfoResponse {
		return nil, 0, fmt.Errorf("unexpected response type: 0x%02x", data[0])
	}

	info, err := parseInfoResponse(data[1:])
	return info, ping, err
}

func parseInfoResponse(data []byte) (*ServerInfo, error) {
	r := bytes.NewReader(data)

	if _, err := r.ReadByte(); err != nil {
		return nil, fmt.Errorf("read protocol version: %w", err)
	}

	name, err := readNullTermString(r)
	if err != nil {
		return nil, fmt.Errorf("read server name: %w", err)
	}

	mapName, err := readNullTermString(r)
	if err != nil {
		return nil, fmt.Errorf("read map: %w", err)
	}

	folder, err := readNullTermString(r)
	if err != nil {
		return nil, fmt.Errorf("read folder: %w", err)
	}

	game, err := readNullTermString(r)
	if err != nil {
		return nil, fmt.Errorf("read game: %w", err)
	}

	// Fixed-size fields after the strings
	var fields struct {
		AppID       uint16
		Players     uint8
		MaxPlayers  uint8
		Bots        uint8
		ServerType  uint8
		Environment uint8
		Visibility  uint8
		VAC         uint8
	}
	if err := binary.Read(r, binary.LittleEndian, &fields); err != nil {
		return nil, fmt.Errorf("read server fields: %w", err)
	}

	version, _ := readNullTermString(r)

	info := &ServerInfo{
		Protocol:    "source",
		Name:        name,
		Map:         mapName,
		Game:        game,
		Players:     int(fields.Players),
		MaxPlayers:  int(fields.MaxPlayers),
		Bots:        int(fields.Bots),
		ServerType:  serverTypeString(fields.ServerType),
		Environment: environmentString(fields.Environment),
		Visibility:  visibilityString(fields.Visibility),
		VAC:         fields.VAC == 1,
		Version:     version,
		Extra:       make(map[string]any),
	}

	if folder != "" {
		info.Extra["folder"] = folder
	}

	// Parse Extra Data Flag (EDF) fields in spec order
	if edf, err := r.ReadByte(); err == nil {
		if edf&0x80 != 0 {
			var edfPort uint16
			if err := binary.Read(r, binary.LittleEndian, &edfPort); err == nil {
				info.ReportedGamePort = edfPort
			}
		}
		if edf&0x10 != 0 {
			var steamID uint64
			if err := binary.Read(r, binary.LittleEndian, &steamID); err == nil {
				info.Extra["steamId"] = steamID
			}
		}
		if edf&0x40 != 0 {
			r.Seek(2, io.SeekCurrent) // skip spectator port (uint16)
			readNullTermString(r)      // skip spectator name
		}
		if edf&0x20 != 0 {
			info.Keywords, _ = readNullTermString(r)
			parseKeywordPlayerCounts(info)
		}
		if edf&0x01 != 0 {
			var gameID uint64
			if err := binary.Read(r, binary.LittleEndian, &gameID); err == nil {
				info.AppID = uint32(gameID & 0xFFFFFF)
			}
		}
	}

	// Fall back to truncated uint16 AppID if EDF GameID wasn't present
	if info.AppID == 0 {
		info.AppID = uint32(fields.AppID)
	}

	return info, nil
}

// challengeQuery performs the A2S challenge-response handshake used by both
// A2S_PLAYER and A2S_RULES. Returns the response payload after the type byte.
func challengeQuery(conn net.Conn, requestByte, responseByte byte) ([]byte, error) {
	challengeReq := []byte{0xFF, 0xFF, 0xFF, 0xFF, requestByte, 0xFF, 0xFF, 0xFF, 0xFF}

	if _, err := conn.Write(challengeReq); err != nil {
		return nil, fmt.Errorf("send challenge: %w", err)
	}

	data, err := readResponse(conn)
	if err != nil {
		return nil, fmt.Errorf("read challenge: %w", err)
	}

	if len(data) >= 5 && data[0] == challengeResponse {
		challenge := data[1:5]
		req := []byte{0xFF, 0xFF, 0xFF, 0xFF, requestByte}
		req = append(req, challenge...)

		if _, err := conn.Write(req); err != nil {
			return nil, fmt.Errorf("send request: %w", err)
		}

		data, err = readResponse(conn)
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
	}

	if len(data) < 2 || data[0] != responseByte {
		return nil, fmt.Errorf("unexpected response type: 0x%02x (expected 0x%02x)", data[0], responseByte)
	}

	return data[1:], nil
}

func queryPlayers(conn net.Conn) ([]PlayerInfo, error) {
	data, err := challengeQuery(conn, a2sPlayerRequest, a2sPlayerResponse)
	if err != nil {
		return nil, fmt.Errorf("a2s_player: %w", err)
	}
	return parsePlayerResponse(data)
}

func parsePlayerResponse(data []byte) ([]PlayerInfo, error) {
	r := bytes.NewReader(data)

	var playerCount uint8
	if err := binary.Read(r, binary.LittleEndian, &playerCount); err != nil {
		return nil, fmt.Errorf("read player count: %w", err)
	}

	players := make([]PlayerInfo, 0, playerCount)

	for i := 0; i < int(playerCount); i++ {
		if _, err := r.ReadByte(); err != nil { // index byte (unused but part of wire format)
			slog.Warn("a2s_player: truncated response", "expected", playerCount, "parsed", i)
			break
		}

		name, err := readNullTermString(r)
		if err != nil {
			slog.Warn("a2s_player: truncated response", "expected", playerCount, "parsed", i)
			break
		}

		var score int32
		if err := binary.Read(r, binary.LittleEndian, &score); err != nil {
			slog.Warn("a2s_player: truncated response", "expected", playerCount, "parsed", i)
			break
		}

		var duration float32
		if err := binary.Read(r, binary.LittleEndian, &duration); err != nil {
			slog.Warn("a2s_player: truncated response", "expected", playerCount, "parsed", i)
			break
		}

		players = append(players, PlayerInfo{
			Name:     name,
			Score:    int(score),
			Duration: Duration{Duration: time.Duration(float64(time.Second) * float64(duration))},
		})
	}

	return players, nil
}

func queryRules(conn net.Conn) (map[string]string, error) {
	data, err := challengeQuery(conn, a2sRulesRequest, a2sRulesResponse)
	if err != nil {
		return nil, fmt.Errorf("a2s_rules: %w", err)
	}
	return parseRulesResponse(data)
}

func parseRulesResponse(data []byte) (map[string]string, error) {
	r := bytes.NewReader(data)

	var ruleCount uint16
	if err := binary.Read(r, binary.LittleEndian, &ruleCount); err != nil {
		return nil, fmt.Errorf("read rule count: %w", err)
	}

	rules := make(map[string]string, ruleCount)

	for i := 0; i < int(ruleCount); i++ {
		name, err := readNullTermString(r)
		if err != nil {
			slog.Warn("a2s_rules: truncated response", "expected", ruleCount, "parsed", i)
			break
		}
		value, err := readNullTermString(r)
		if err != nil {
			slog.Warn("a2s_rules: truncated response", "expected", ruleCount, "parsed", i)
			break
		}
		rules[name] = value
	}

	return rules, nil
}

// readNullTermString reads a null-terminated string from a bytes.Reader.
func readNullTermString(r *bytes.Reader) (string, error) {
	var result []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return string(result), err
		}
		if b == 0 {
			return string(result), nil
		}
		result = append(result, b)
	}
}

func serverTypeString(b uint8) string {
	switch b {
	case 'd':
		return "dedicated"
	case 'l':
		return "listen"
	case 'p':
		return "proxy"
	default:
		return "unknown"
	}
}

func environmentString(b uint8) string {
	switch b {
	case 'l':
		return "linux"
	case 'w':
		return "windows"
	case 'm', 'o':
		return "mac"
	default:
		return "unknown"
	}
}

func visibilityString(b uint8) string {
	if b == 0 {
		return "public"
	}
	return "private"
}

// parseKeywordPlayerCounts parses cp (current players) and mp (max players)
// tags from the keywords string. Rust servers use these because the A2S_INFO
// binary fields are uint8 and overflow at 255.
func parseKeywordPlayerCounts(info *ServerInfo) {
	for _, tag := range strings.Split(info.Keywords, ",") {
		tag = strings.TrimSpace(tag)
		if v, ok := strings.CutPrefix(tag, "cp"); ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				slog.Debug("a2s: keyword tag override", "field", "players", "old", info.Players, "new", n)
				info.Players = n
			}
		} else if v, ok := strings.CutPrefix(tag, "mp"); ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				slog.Debug("a2s: keyword tag override", "field", "maxPlayers", "old", info.MaxPlayers, "new", n)
				info.MaxPlayers = n
			}
		}
	}
}
