package gjq

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net"
	"sync"
	"time"

	"github.com/warsmite/gamejanitor/games"
	"github.com/warsmite/gjq/protocol"
)

type Duration = protocol.Duration
type ServerInfo = protocol.ServerInfo
type PlayerInfo = protocol.PlayerInfo

type QueryOptions struct {
	Game            string
	Protocol        string
	Timeout         time.Duration
	Players         bool
	Rules           bool
	Direct          bool // treat port as the exact query port, skip port derivation
	EOSClientID     string
	EOSClientSecret string
}

type DiscoverOptions struct {
	Timeout    time.Duration
	PortRanges []PortRange
	Players    bool
}

type PortRange struct {
	Start uint16
	End   uint16
}

type candidate struct {
	port     uint16
	protocol string
	priority int // lower = better; 0 = best possible match
}

// Registry is the shared game registry loaded from gamejanitor's embedded game data.
var Registry *games.Registry

func init() {
	var err error
	Registry, err = games.NewRegistry()
	if err != nil {
		panic(fmt.Sprintf("gjq: failed to load game registry: %v", err))
	}
}

// hasSupport checks if a game's query config supports a feature.
func hasSupport(g *games.GameDef, feature string) bool {
	if g.Query == nil {
		return false
	}
	for _, s := range g.Query.Supports {
		if s == feature {
			return true
		}
	}
	return false
}

// eosConfig maps a GameDef's EOS query config to the protocol layer's EOSConfig.
func eosConfig(g *games.GameDef) *protocol.EOSConfig {
	if g.Query == nil || g.Query.EOS == nil {
		return nil
	}
	e := g.Query.EOS
	return &protocol.EOSConfig{
		ClientID:        e.ClientID,
		ClientSecret:    e.ClientSecret,
		DeploymentID:    e.DeploymentID,
		UseExternalAuth: e.UseExternalAuth,
		UseWildcard:     e.UseWildcard,
		Attributes:      e.Attributes,
	}
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

	if opts.Protocol != "" {
		if _, err := protocol.Get(opts.Protocol); err != nil {
			return nil, fmt.Errorf("unknown protocol %q", opts.Protocol)
		}
	}

	queryOpts := protocol.QueryOpts{Players: opts.Players, Rules: opts.Rules, ResolvedIP: resolvedIP}

	var gc *games.GameDef
	if opts.Game != "" {
		gc = Registry.Get(opts.Game)
		if gc == nil || !gc.HasQuery() {
			return nil, fmt.Errorf("unknown game %q — run 'gjq games' to see supported games", opts.Game)
		}
	}

	if gc != nil {
		if opts.Players && !hasSupport(gc, "players") {
			return nil, fmt.Errorf("%s does not support --players", gc.Name)
		}
		if opts.Rules && !hasSupport(gc, "rules") {
			return nil, fmt.Errorf("%s does not support --rules", gc.Name)
		}

		if eosCfg := eosConfig(gc); eosCfg != nil {
			if opts.EOSClientID != "" {
				eosCfg.ClientID = opts.EOSClientID
			}
			if opts.EOSClientSecret != "" {
				eosCfg.ClientSecret = opts.EOSClientSecret
			}
			queryOpts.EOS = eosCfg
		}
	}

	candidates := buildCandidates(port, gc, opts.Direct, opts.Protocol)
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

// autoDetectProtocols returns protocols suitable for blind probing.
// EOS requires game-specific credentials, fivem and tshock are HTTP APIs
// that shouldn't be used for auto-detection on unknown ports.
func autoDetectProtocols() []string {
	skip := map[string]bool{"eos": true, "fivem": true, "tshock": true}
	var names []string
	for name := range protocol.All() {
		if !skip[name] {
			names = append(names, name)
		}
	}
	return names
}

// buildCandidates generates prioritized (port, protocol) pairs to try concurrently.
func buildCandidates(port uint16, gc *games.GameDef, direct bool, proto string) []candidate {
	// --protocol: user specifies exact protocol and query port
	if proto != "" {
		return []candidate{{port: port, protocol: proto, priority: 0}}
	}

	// --direct: user guarantees this is the query port
	if direct {
		if gc != nil {
			return []candidate{{port: port, protocol: gc.Query.Protocol, priority: 0}}
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

// candidatesForPort returns a candidate per auto-detect protocol for the given port and priority.
func candidatesForPort(port uint16, priority int) []candidate {
	protos := autoDetectProtocols()
	candidates := make([]candidate, len(protos))
	for i, name := range protos {
		candidates[i] = candidate{port: port, protocol: name, priority: priority}
	}
	return candidates
}

func buildCandidatesWithGame(port uint16, gc *games.GameDef) []candidate {
	// No port given is handled by CLI (fills in defaultQueryPort), so port is always set here.
	queryPort := gc.QueryPort()
	gamePort := gc.GamePort()
	proto := gc.Query.Protocol

	if port == queryPort {
		return []candidate{{port: port, protocol: proto, priority: 0}}
	}

	if port == gamePort && queryPort != gamePort {
		return []candidate{
			{port: queryPort, protocol: proto, priority: 0},
			{port: port, protocol: proto, priority: 1},
		}
	}

	// Arbitrary port — try user's port, then offset-derived in both directions
	candidates := []candidate{{port: port, protocol: proto, priority: 0}}
	if queryPort != gamePort {
		offset := int(queryPort) - int(gamePort)
		for _, d := range []int{int(port) + offset, int(port) - offset} {
			if d > 0 && d <= 65535 && uint16(d) != port {
				candidates = append(candidates, candidate{port: uint16(d), protocol: proto, priority: 1})
			}
		}
	}
	return candidates
}

func buildCandidatesAutoDetect(port uint16) []candidate {
	// Priority 0: user's port with all non-EOS protocols
	candidates := candidatesForPort(port, 0)

	// Priority 1: for games where user's port is a known game port, add the query port
	for _, g := range Registry.ByGamePort(port) {
		if !g.HasQuery() {
			continue
		}
		queryPort := g.QueryPort()
		gamePort := g.GamePort()
		if queryPort != gamePort {
			candidates = append(candidates, candidate{port: queryPort, protocol: g.Query.Protocol, priority: 1})
		}
	}

	// Priority 2: offset-derived ports from all games with differing query/game ports
	seen := map[uint16]bool{port: true}
	for _, g := range Registry.WithQuery() {
		queryPort := g.QueryPort()
		gamePort := g.GamePort()
		if queryPort == gamePort {
			continue
		}
		offset := int(queryPort) - int(gamePort)
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
		for _, g := range Registry.WithQuery() {
			ports = append(ports, g.QueryPort())
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
				info, err := raceQueryPriority(probeCtx, address, buildCandidates(port, nil, true, ""), queryOpts)
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
func enrichResult(info *ServerInfo, gc *games.GameDef) {
	if gc == nil {
		if info.AppID != 0 {
			gc = Registry.ByAppID(info.AppID)
		} else if info.Protocol == "minecraft" {
			gc = Registry.Get("minecraft-java")
		}
	}

	if gc != nil {
		info.Game = gc.Name
		if gc.Query != nil && gc.Query.Notes != "" {
			if info.Extra == nil {
				info.Extra = make(map[string]any)
			}
			info.Extra["gameNotes"] = gc.Query.Notes
		}
	}
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
