package gsq

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strings"
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

	resolvedIP, err := resolveHost(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", address, err)
	}

	queryOpts := protocol.QueryOpts{Players: opts.Players, ResolvedIP: resolvedIP}

	if opts.Game != "" {
		return queryByGame(ctx, address, port, opts.Game, queryOpts)
	}

	var attempts []attempt
	for name := range protocol.All() {
		attempts = append(attempts, attempt{port: port, protocol: name})
	}

	info, err := raceQuery(ctx, address, attempts, queryOpts)
	if err != nil {
		return nil, fmt.Errorf("no protocol matched for %s:%d: %w", address, port, err)
	}
	inferGamePort(info)
	resolveGameName(info)
	return info, nil
}

func queryByGame(ctx context.Context, address string, givenPort uint16, game string, queryOpts protocol.QueryOpts) (*ServerInfo, error) {
	gc := LookupGame(game)
	if gc == nil {
		return nil, fmt.Errorf("unknown game %q — run 'gsq games' to see supported games", game)
	}

	var candidatePorts []uint16
	if gc.DefaultQueryPort >= gc.DefaultGamePort {
		offset := gc.DefaultQueryPort - gc.DefaultGamePort
		candidatePorts = dedupPorts(givenPort, givenPort+offset, gc.DefaultQueryPort)
	} else {
		candidatePorts = dedupPorts(givenPort, gc.DefaultQueryPort)
	}

	var attempts []attempt
	for _, p := range candidatePorts {
		attempts = append(attempts, attempt{port: p, protocol: gc.Protocol})
	}

	info, err := raceQuery(ctx, address, attempts, queryOpts)
	if err != nil {
		return nil, fmt.Errorf("no query port worked for %s (game %s): %w", address, game, err)
	}

	// User gave a game port; query may have succeeded on a different (query) port
	info.QueryPort = info.GamePort
	info.GamePort = givenPort
	info.Game = gc.Name
	sanitizeInfo(info)

	return info, nil
}

type attempt struct {
	port     uint16
	protocol string
}

// raceQuery tries all attempts concurrently, returning the first successful result.
func raceQuery(ctx context.Context, address string, attempts []attempt, queryOpts protocol.QueryOpts) (*ServerInfo, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		info *ServerInfo
		err  error
	}

	resultCh := make(chan result, len(attempts))
	var wg sync.WaitGroup

	for _, a := range attempts {
		wg.Add(1)
		go func(a attempt) {
			defer wg.Done()
			info, err := queryWithProtocol(ctx, address, a.port, a.protocol, queryOpts)
			if err != nil {
				resultCh <- result{err: err}
				return
			}
			resultCh <- result{info: info}
		}(a)
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
	return nil, lastErr
}

func queryWithProtocol(ctx context.Context, address string, port uint16, protoName string, queryOpts protocol.QueryOpts) (*ServerInfo, error) {
	q, err := protocol.Get(protoName)
	if err != nil {
		return nil, fmt.Errorf("get protocol %q: %w", protoName, err)
	}

	slog.Debug("querying server", "protocol", protoName, "address", address, "port", port)
	info, err := q.Query(ctx, address, port, queryOpts)
	if err != nil {
		slog.Debug("query failed", "protocol", protoName, "address", address, "port", port, "error", err)
		return nil, err
	}
	slog.Debug("query succeeded", "protocol", protoName, "address", address, "port", port)
	return info, nil
}

// Discover scans a host for game servers by probing known default ports
// or a custom port range with all registered protocols.
func Discover(ctx context.Context, address string, opts DiscoverOptions) ([]*ServerInfo, error) {
	resolvedIP, err := resolveHost(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", address, err)
	}

	queryOpts := protocol.QueryOpts{Players: opts.Players, ResolvedIP: resolvedIP}

	ports := collectPorts(opts.PortRanges)
	protocols := protocol.All()

	var probes []attempt
	for _, port := range ports {
		for name := range protocols {
			probes = append(probes, attempt{port: port, protocol: name})
		}
	}

	resultCh := make(chan *ServerInfo, len(probes))
	sem := make(chan struct{}, 20)

	for _, p := range probes {
		go func(p attempt) {
			sem <- struct{}{}
			defer func() { <-sem }()

			probeCtx := ctx
			if opts.Timeout > 0 {
				var cancel context.CancelFunc
				probeCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
				defer cancel()
			}

			info, err := queryWithProtocol(probeCtx, address, p.port, p.protocol, queryOpts)
			if err != nil {
				resultCh <- nil
				return
			}
			inferGamePort(info)
			resolveGameName(info)
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

func collectPorts(portRanges []PortRange) []uint16 {
	var ports []uint16

	if len(portRanges) > 0 {
		for _, pr := range portRanges {
			for port := pr.Start; port <= pr.End; port++ {
				ports = append(ports, port)
			}
		}
	} else {
		for _, g := range SupportedGames() {
			ports = append(ports, g.DefaultQueryPort, g.DefaultGamePort)
		}
	}

	return dedupPorts(ports...)
}

// inferGamePort identifies the game via AppID and infers the game port from the
// known query-to-game port offset. Used by auto-detect Query and Discover.
func inferGamePort(info *ServerInfo) {
	queriedPort := info.GamePort

	var gc *GameConfig
	if info.AppID != 0 {
		gc = LookupGameByAppID(info.AppID)
	} else if info.Protocol == "minecraft" {
		gc = LookupGame("minecraft")
	}

	if gc != nil && gc.DefaultQueryPort > gc.DefaultGamePort {
		offset := gc.DefaultQueryPort - gc.DefaultGamePort
		if queriedPort >= offset {
			info.GamePort = queriedPort - offset
			info.QueryPort = queriedPort
		}
	}
}

var tagRegex = regexp.MustCompile(`<[^>]+>`)

func sanitize(s string) string {
	return strings.TrimSpace(tagRegex.ReplaceAllString(s, ""))
}

func sanitizeInfo(info *ServerInfo) {
	info.Name = sanitize(info.Name)
	info.Map = sanitize(info.Map)
}

// resolveGameName sets the Game field from our registry if the AppID matches,
// otherwise sanitizes the server-reported value.
func resolveGameName(info *ServerInfo) {
	var gc *GameConfig
	if info.AppID != 0 {
		gc = LookupGameByAppID(info.AppID)
	} else if info.Protocol == "minecraft" {
		gc = LookupGame("minecraft")
	}

	if gc != nil {
		info.Game = gc.Name
	} else {
		info.Game = sanitize(info.Game)
	}
	sanitizeInfo(info)
}

// resolveHost resolves a hostname to an IP once so concurrent queries don't repeat DNS.
// If the address is already an IP, it returns it as-is.
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
