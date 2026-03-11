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

func queryByGame(ctx context.Context, address string, gamePort uint16, game string) (*ServerInfo, error) {
	gc := LookupGame(game)
	if gc == nil {
		return nil, fmt.Errorf("unknown game %q — run 'gsq games' to see supported games", game)
	}

	queryPort := gamePort + (gc.DefaultQueryPort - gc.DefaultGamePort)

	info, err := queryWithProtocol(ctx, address, queryPort, gc.Protocol)
	if err != nil {
		return nil, err
	}

	info.Port = gamePort
	return info, nil
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

	type probe struct {
		queryPort uint16
		gamePort  uint16
		protocol  string
	}

	var probes []probe

	if len(opts.PortRanges) > 0 {
		for _, pr := range opts.PortRanges {
			for port := pr.Start; port <= pr.End; port++ {
				for name := range protocol.All() {
					probes = append(probes, probe{queryPort: port, gamePort: port, protocol: name})
				}
			}
		}
	} else {
		seen := make(map[string]bool)
		for _, g := range SupportedGames() {
			key := fmt.Sprintf("%s:%d", g.Protocol, g.DefaultQueryPort)
			if seen[key] {
				continue
			}
			seen[key] = true
			probes = append(probes, probe{queryPort: g.DefaultQueryPort, gamePort: g.DefaultGamePort, protocol: g.Protocol})
		}
	}

	resultCh := make(chan *ServerInfo, len(probes))
	sem := make(chan struct{}, 20)

	for _, p := range probes {
		go func(p probe) {
			sem <- struct{}{}
			defer func() { <-sem }()
			info, err := queryWithProtocol(ctx, address, p.queryPort, p.protocol)
			if err != nil {
				slog.Debug("scan miss", "port", p.queryPort, "protocol", p.protocol, "error", err)
				resultCh <- nil
				return
			}
			info.Port = p.gamePort
			slog.Debug("scan hit", "port", p.queryPort, "protocol", p.protocol)
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
