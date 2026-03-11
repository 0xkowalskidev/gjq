package gsq

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/0xkowalskidev/gsq/internal/protocol"

	// Register protocol implementations
	_ "github.com/0xkowalskidev/gsq/internal/protocol/minecraft"
	_ "github.com/0xkowalskidev/gsq/internal/protocol/source"
)

// Query queries a game server at the given address and port.
// If opts.Game is set, it uses the corresponding protocol directly.
// Otherwise, it auto-detects by trying all protocols concurrently.
func Query(ctx context.Context, address string, port uint16, opts QueryOptions) (*ServerInfo, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	if opts.Game != "" {
		return queryByGame(ctx, address, port, opts.Game)
	}

	return detectProtocol(ctx, address, port)
}

func queryByGame(ctx context.Context, address string, givenPort uint16, game string) (*ServerInfo, error) {
	gc := LookupGame(game)
	if gc == nil {
		return nil, fmt.Errorf("unknown game %q — run 'gsq games' to see supported games", game)
	}

	offset := gc.DefaultQueryPort - gc.DefaultGamePort

	seen := make(map[uint16]bool)
	var queryPorts []uint16

	addCandidate := func(qp uint16) {
		if !seen[qp] {
			seen[qp] = true
			queryPorts = append(queryPorts, qp)
		}
	}

	addCandidate(givenPort)                // Maybe the user provided the query port directly
	addCandidate(givenPort + offset)       // User provided the game port
	addCandidate(gc.DefaultQueryPort)      // Custom game port but default query port

	type result struct {
		info      *ServerInfo
		queryPort uint16
		err       error
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultCh := make(chan result, len(queryPorts))
	var wg sync.WaitGroup

	for _, qp := range queryPorts {
		wg.Add(1)
		go func(qp uint16) {
			defer wg.Done()
			slog.Debug("trying query port candidate", "queryPort", qp, "game", game)
			info, err := queryWithProtocol(ctx, address, qp, gc.Protocol)
			if err != nil {
				slog.Debug("candidate failed", "queryPort", qp, "error", err)
				resultCh <- result{err: err}
				return
			}
			slog.Debug("candidate succeeded", "queryPort", qp)
			resultCh <- result{info: info, queryPort: qp}
		}(qp)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var lastErr error
	for r := range resultCh {
		if r.err != nil {
			lastErr = r.err
			continue
		}
		cancel()
		r.info.Port = resolveGamePort(r.info, r.queryPort, givenPort, gc)
		r.info.GamePort = 0
		return r.info, nil
	}

	return nil, fmt.Errorf("no query port worked for %s (game %s): %w", address, game, lastErr)
}

func resolveGamePort(info *ServerInfo, matchedQueryPort, givenPort uint16, gc *GameConfig) uint16 {
	// Prefer the protocol-reported game port when available (e.g. from A2S_INFO EDF)
	if info.GamePort != 0 {
		return info.GamePort
	}

	offset := gc.DefaultQueryPort - gc.DefaultGamePort

	switch matchedQueryPort {
	case givenPort + offset:
		// Offset candidate matched — user gave the game port
		return givenPort
	case givenPort:
		// Direct candidate matched — user likely gave the query port
		return givenPort - offset
	case gc.DefaultQueryPort:
		// Default query port matched — use the default game port
		return gc.DefaultGamePort
	default:
		return givenPort
	}
}

func queryWithProtocol(ctx context.Context, address string, port uint16, protoName string) (*ServerInfo, error) {
	q, err := protocol.Get(protoName)
	if err != nil {
		return nil, fmt.Errorf("get protocol %q: %w", protoName, err)
	}

	slog.Debug("querying server", "protocol", protoName, "address", address, "port", port)
	return q.Query(ctx, address, port)
}

// detectProtocol tries all registered protocols concurrently and returns the first valid response.
func detectProtocol(ctx context.Context, address string, port uint16) (*ServerInfo, error) {
	all := protocol.All()
	if len(all) == 0 {
		return nil, fmt.Errorf("no protocols registered")
	}

	type result struct {
		info *ServerInfo
		err  error
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultCh := make(chan result, len(all))
	var wg sync.WaitGroup

	for name, querier := range all {
		wg.Add(1)
		go func(name string, q protocol.Querier) {
			defer wg.Done()
			slog.Debug("trying protocol", "protocol", name, "address", address, "port", port)
			info, err := q.Query(ctx, address, port)
			if err != nil {
				slog.Debug("protocol failed", "protocol", name, "error", err)
				resultCh <- result{err: err}
				return
			}
			slog.Debug("protocol succeeded", "protocol", name)
			resultCh <- result{info: info}
		}(name, querier)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var lastErr error
	for r := range resultCh {
		if r.err != nil {
			lastErr = r.err
			continue
		}
		cancel()
		return r.info, nil
	}

	return nil, fmt.Errorf("no protocol matched for %s:%d: %w", address, port, lastErr)
}

// Discover scans a host for game servers by probing known default ports
// or a custom port range with all registered protocols.
func Discover(ctx context.Context, address string, opts DiscoverOptions) ([]*ServerInfo, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	ports := collectPorts(opts.PortRanges)
	protocols := protocol.All()

	type probe struct {
		port     uint16
		protocol string
	}

	var probes []probe
	for _, port := range ports {
		for name := range protocols {
			probes = append(probes, probe{port: port, protocol: name})
		}
	}

	resultCh := make(chan *ServerInfo, len(probes))
	sem := make(chan struct{}, 20)

	for _, p := range probes {
		go func(p probe) {
			sem <- struct{}{}
			defer func() { <-sem }()
			info, err := queryWithProtocol(ctx, address, p.port, p.protocol)
			if err != nil {
				slog.Debug("scan miss", "port", p.port, "protocol", p.protocol, "error", err)
				resultCh <- nil
				return
			}
			if info.GamePort != 0 {
				info.Port = info.GamePort
				info.GamePort = 0
			}
			slog.Debug("scan hit", "port", p.port, "protocol", p.protocol)
			resultCh <- info
		}(p)
	}

	var servers []*ServerInfo
	for range probes {
		if info := <-resultCh; info != nil {
			servers = append(servers, info)
		}
	}

	return servers, nil
}

// collectPorts returns a deduplicated list of ports to probe.
// If port ranges are provided, uses those. Otherwise, collects all
// unique default ports from the game registry.
func collectPorts(portRanges []PortRange) []uint16 {
	if len(portRanges) > 0 {
		seen := make(map[uint16]bool)
		var ports []uint16
		for _, pr := range portRanges {
			for port := pr.Start; port <= pr.End; port++ {
				if !seen[port] {
					seen[port] = true
					ports = append(ports, port)
				}
			}
		}
		return ports
	}

	seen := make(map[uint16]bool)
	var ports []uint16
	for _, g := range SupportedGames() {
		for _, port := range []uint16{g.DefaultQueryPort, g.DefaultGamePort} {
			if !seen[port] {
				seen[port] = true
				ports = append(ports, port)
			}
		}
	}
	return ports
}
