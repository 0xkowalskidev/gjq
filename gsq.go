package gsq

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net"
	"regexp"
	"strings"
	"sync"

	"github.com/0xkowalskidev/gsq/internal/protocol"

	// Register protocol implementations
	_ "github.com/0xkowalskidev/gsq/internal/protocol/eos"
	_ "github.com/0xkowalskidev/gsq/internal/protocol/minecraft"
	_ "github.com/0xkowalskidev/gsq/internal/protocol/raknet"
	_ "github.com/0xkowalskidev/gsq/internal/protocol/source"
	_ "github.com/0xkowalskidev/gsq/internal/protocol/tshock"
)

type candidate struct {
	port     uint16
	protocol string
	priority int // lower = better; 0 = best possible match
}

// Query queries a game server at the given address and port.
// All candidate (port, protocol) combinations are probed concurrently.
// The best result (lowest priority) is returned, with early exit when
// a priority-0 candidate succeeds.
func Query(ctx context.Context, address string, port uint16, opts QueryOptions) (*ServerInfo, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	resolvedIP, err := resolveHost(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", address, err)
	}

	queryOpts := protocol.QueryOpts{Players: opts.Players, ResolvedIP: resolvedIP}

	var gc *GameConfig
	if opts.Game != "" {
		gc = LookupGame(opts.Game)
		if gc == nil {
			return nil, fmt.Errorf("unknown game %q — run 'gsq games' to see supported games", opts.Game)
		}
	}

	if gc != nil {
		if eosCfg, ok := gc.ProtocolConfig.(*protocol.EOSConfig); ok {
			cfg := *eosCfg
			if opts.EOSClientID != "" {
				cfg.ClientID = opts.EOSClientID
			}
			if opts.EOSClientSecret != "" {
				cfg.ClientSecret = opts.EOSClientSecret
			}
			queryOpts.EOS = &cfg
		}
	}

	candidates := buildCandidates(port, gc, opts.Direct)
	for _, c := range candidates {
		slog.Debug("candidate", "port", c.port, "protocol", c.protocol, "priority", c.priority)
	}

	info, err := raceQueryPriority(ctx, address, candidates, queryOpts)
	if err != nil {
		if gc != nil {
			return nil, fmt.Errorf("no query port worked for %s (game %s): %w", address, opts.Game, err)
		}
		return nil, fmt.Errorf("no protocol matched for %s:%d: %w", address, port, err)
	}

	info.Address = address
	// GamePort = user's input port. Protocol-reported game ports are unreliable
	// (containerized servers remap ports), and offset-based guessing is wrong for
	// non-standard layouts. The user's port is the most useful value here.
	info.GamePort = port
	enrichResult(info, gc)
	return info, nil
}

// nonEOSProtocols returns all registered protocol names except EOS,
// which requires game-specific credentials and can't be used for auto-detection.
func nonEOSProtocols() []string {
	var names []string
	for name := range protocol.All() {
		if name != "eos" {
			names = append(names, name)
		}
	}
	return names
}

// buildCandidates generates prioritized (port, protocol) pairs to try concurrently.
func buildCandidates(port uint16, gc *GameConfig, direct bool) []candidate {
	// --direct: user guarantees this is the query port
	if direct {
		if gc != nil {
			return []candidate{{port: port, protocol: gc.Protocol, priority: 0}}
		}
		return candidatesForPort(port, 0)
	}

	// --game set: we know the protocol
	if gc != nil {
		return buildCandidatesWithGame(port, gc)
	}

	// Auto-detect: no game specified
	return buildCandidatesAutoDetect(port)
}

// candidatesForPort returns a candidate per non-EOS protocol for the given port and priority.
func candidatesForPort(port uint16, priority int) []candidate {
	protos := nonEOSProtocols()
	candidates := make([]candidate, len(protos))
	for i, name := range protos {
		candidates[i] = candidate{port: port, protocol: name, priority: priority}
	}
	return candidates
}

func buildCandidatesWithGame(port uint16, gc *GameConfig) []candidate {
	// No port given is handled by CLI (fills in defaultQueryPort), so port is always set here.
	if port == gc.DefaultQueryPort {
		return []candidate{{port: port, protocol: gc.Protocol, priority: 0}}
	}

	if port == gc.DefaultGamePort && gc.DefaultQueryPort != gc.DefaultGamePort {
		return []candidate{
			{port: gc.DefaultQueryPort, protocol: gc.Protocol, priority: 0},
			{port: port, protocol: gc.Protocol, priority: 1},
		}
	}

	// Arbitrary port — try user's port, then offset-derived in both directions
	candidates := []candidate{{port: port, protocol: gc.Protocol, priority: 0}}
	if gc.DefaultQueryPort != gc.DefaultGamePort {
		offset := int(gc.DefaultQueryPort) - int(gc.DefaultGamePort)
		for _, d := range []int{int(port) + offset, int(port) - offset} {
			if d > 0 && d <= 65535 && uint16(d) != port {
				candidates = append(candidates, candidate{port: uint16(d), protocol: gc.Protocol, priority: 1})
			}
		}
	}
	return candidates
}

func buildCandidatesAutoDetect(port uint16) []candidate {
	// Priority 0: user's port with all non-EOS protocols
	candidates := candidatesForPort(port, 0)

	// Priority 1: for games where user's port is a known game port, add the query port
	for _, g := range GamesWithGamePort(port) {
		if g.DefaultQueryPort != g.DefaultGamePort {
			candidates = append(candidates, candidate{port: g.DefaultQueryPort, protocol: g.Protocol, priority: 1})
		}
	}

	// Priority 2: offset-derived ports from all games with differing query/game ports
	seen := map[uint16]bool{port: true}
	for _, g := range SupportedGames() {
		if g.DefaultQueryPort == g.DefaultGamePort {
			continue
		}
		offset := int(g.DefaultQueryPort) - int(g.DefaultGamePort)
		derived := int(port) + offset
		if derived > 0 && derived <= 65535 && !seen[uint16(derived)] {
			seen[uint16(derived)] = true
			candidates = append(candidates, candidatesForPort(uint16(derived), 2)...)
		}
	}

	return candidates
}

// raceQueryPriority fires all candidates concurrently and returns the best result.
// Returns immediately when a priority-0 candidate succeeds. Otherwise waits for all
// candidates to complete (or context to expire) and returns the lowest-priority success.
func raceQueryPriority(ctx context.Context, address string, candidates []candidate, queryOpts protocol.QueryOpts) (*ServerInfo, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		info     *ServerInfo
		err      error
		priority int
	}

	resultCh := make(chan result, len(candidates))

	var wg sync.WaitGroup
	for _, c := range candidates {
		wg.Add(1)
		go func(c candidate) {
			defer wg.Done()

			q, err := protocol.Get(c.protocol)
			if err != nil {
				resultCh <- result{err: fmt.Errorf("get protocol %q: %w", c.protocol, err), priority: c.priority}
				return
			}

			slog.Debug("querying server", "protocol", c.protocol, "address", address, "port", c.port, "priority", c.priority)
			info, err := q.Query(ctx, address, c.port, queryOpts)
			if err != nil {
				slog.Debug("query failed", "protocol", c.protocol, "address", address, "port", c.port, "error", err)
				resultCh <- result{err: err, priority: c.priority}
				return
			}
			slog.Debug("query succeeded", "protocol", c.protocol, "address", address, "port", c.port, "priority", c.priority)
			resultCh <- result{info: info, priority: c.priority}
		}(c)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var bestInfo *ServerInfo
	bestPriority := math.MaxInt
	var lastErr error

	for r := range resultCh {
		if r.err != nil {
			lastErr = r.err
			continue
		}
		if r.priority < bestPriority {
			bestInfo = r.info
			bestPriority = r.priority
		}
		if bestPriority == 0 {
			cancel()
			return bestInfo, nil
		}
	}

	if bestInfo != nil {
		return bestInfo, nil
	}
	return nil, lastErr
}

// Discover scans a host for game servers by probing known default query ports
// or a custom port range with all registered protocols.
func Discover(ctx context.Context, address string, opts DiscoverOptions) ([]*ServerInfo, error) {
	resolvedIP, err := resolveHost(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", address, err)
	}

	var ports []uint16
	if len(opts.PortRanges) > 0 {
		for _, pr := range opts.PortRanges {
			for port := uint32(pr.Start); port <= uint32(pr.End); port++ {
				ports = append(ports, uint16(port))
			}
		}
	} else {
		for _, g := range SupportedGames() {
			ports = append(ports, g.DefaultQueryPort)
		}
	}
	ports = dedupPorts(ports...)

	queryOpts := protocol.QueryOpts{Players: opts.Players, ResolvedIP: resolvedIP}

	workers := 256
	portCh := make(chan uint16, workers)
	resultCh := make(chan *ServerInfo, workers)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range portCh {
				probeCtx, probeCancel := context.WithTimeout(ctx, opts.Timeout)
				info, err := raceQueryPriority(probeCtx, address, buildCandidates(port, nil, true), queryOpts)
				probeCancel()
				if err != nil {
					continue
				}

				info.Address = address
				enrichResult(info, nil)
				resultCh <- info
			}
		}()
	}

	go func() {
		for _, port := range ports {
			portCh <- port
		}
		close(portCh)
	}()

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	seen := make(map[string]bool)
	var servers []*ServerInfo
	for info := range resultCh {
		key := fmt.Sprintf("%s:%d", info.Protocol, info.QueryPort)
		if seen[key] {
			continue
		}
		seen[key] = true
		servers = append(servers, info)
	}

	return servers, nil
}

// enrichResult sets game/query ports and game name from the GameConfig.
// If gc is nil, it attempts to look up by AppID.
func enrichResult(info *ServerInfo, gc *GameConfig) {
	if gc == nil {
		if info.AppID != 0 {
			gc = LookupGameByAppID(info.AppID)
		} else if info.Protocol == "minecraft" {
			gc = LookupGame("minecraft-java")
		}
	}

	if gc != nil {
		info.Game = gc.Name
	} else {
		info.Game = sanitize(info.Game)
	}
	info.Name = sanitize(info.Name)
	info.Map = sanitize(info.Map)
	info.Version = sanitize(info.Version)
	info.ServerType = sanitize(info.ServerType)
	info.Environment = sanitize(info.Environment)
	info.Visibility = sanitize(info.Visibility)
	for i := range info.PlayerList {
		info.PlayerList[i].Name = sanitize(info.PlayerList[i].Name)
	}
}

var sanitizeRe = regexp.MustCompile(strings.Join([]string{
	`§[0-9a-fk-or]`,  // Minecraft color/formatting codes
	`\x1b\[[0-9;]*m`, // ANSI escape sequences
	`<[^>]+>`,         // HTML/Unity rich text tags
	`\^[0-9]`,         // Quake-style color codes
	`[\x00-\x1f]`,     // Control characters
	`\s{2,}`,          // Collapse runs of whitespace
}, "|"))

func sanitize(s string) string {
	return strings.TrimSpace(sanitizeRe.ReplaceAllStringFunc(s, func(m string) string {
		if m[0] == ' ' || m[0] == '\t' {
			return " "
		}
		return ""
	}))
}

func resolveHost(ctx context.Context, address string) (string, error) {
	if net.ParseIP(address) != nil {
		return address, nil
	}
	ips, err := net.DefaultResolver.LookupHost(ctx, address)
	if err != nil {
		return "", err
	}
	slog.Debug("resolved host", "host", address, "ip", ips[0])
	return ips[0], nil
}

func dedupPorts(ports ...uint16) []uint16 {
	seen := make(map[uint16]bool, len(ports))
	var result []uint16
	for _, p := range ports {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	return result
}
