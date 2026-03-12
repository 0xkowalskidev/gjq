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

	var gc *GameConfig
	if opts.Game != "" {
		gc = LookupGame(opts.Game)
		if gc == nil {
			return nil, fmt.Errorf("unknown game %q — run 'gsq games' to see supported games", opts.Game)
		}
	}

	candidatePorts := buildCandidatePorts(port, gc)

	var attempts []attempt
	if gc != nil {
		for _, p := range candidatePorts {
			attempts = append(attempts, attempt{port: p, protocol: gc.Protocol})
		}
	} else {
		for _, p := range candidatePorts {
			for name := range protocol.All() {
				attempts = append(attempts, attempt{port: p, protocol: name})
			}
		}
	}

	info, err := raceQuery(ctx, address, attempts, queryOpts)
	if err != nil {
		if gc != nil {
			return nil, fmt.Errorf("no query port worked for %s (game %s): %w", address, opts.Game, err)
		}
		return nil, fmt.Errorf("no protocol matched for %s:%d: %w", address, port, err)
	}

	enrichResult(info, gc)
	return info, nil
}

// Discover scans a host for game servers by probing known default ports
// or a custom port range with all registered protocols.
func Discover(ctx context.Context, address string, opts DiscoverOptions) ([]*ServerInfo, error) {
	ports := collectPorts(opts.PortRanges)

	workers := 256
	portCh := make(chan uint16, workers)
	resultCh := make(chan *ServerInfo, workers)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range portCh {
				info, err := Query(ctx, address, port, QueryOptions{
					Timeout: opts.Timeout,
					Players: opts.Players,
				})
				if err != nil {
					continue
				}
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

			q, err := protocol.Get(a.protocol)
			if err != nil {
				resultCh <- result{err: fmt.Errorf("get protocol %q: %w", a.protocol, err)}
				return
			}

			slog.Debug("querying server", "protocol", a.protocol, "address", address, "port", a.port)
			info, err := q.Query(ctx, address, a.port, queryOpts)
			if err != nil {
				slog.Debug("query failed", "protocol", a.protocol, "address", address, "port", a.port, "error", err)
				resultCh <- result{err: err}
				return
			}
			slog.Debug("query succeeded", "protocol", a.protocol, "address", address, "port", a.port)
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

func buildCandidatePorts(givenPort uint16, gc *GameConfig) []uint16 {
	ports := []uint16{givenPort}
	if gc != nil {
		ports = append(ports, gc.DefaultQueryPort)
		if gc.DefaultQueryPort >= gc.DefaultGamePort {
			offset := gc.DefaultQueryPort - gc.DefaultGamePort
			ports = append(ports, givenPort+offset)
		}
	} else {
		for _, g := range SupportedGames() {
			if g.DefaultQueryPort > g.DefaultGamePort {
				offset := g.DefaultQueryPort - g.DefaultGamePort
				ports = append(ports, givenPort+offset)
			}
		}
	}
	return dedupPorts(ports...)
}

// enrichResult sets game/query ports and game name from the GameConfig.
// If gc is nil, it attempts to look up by AppID.
func enrichResult(info *ServerInfo, gc *GameConfig) {
	if gc == nil {
		if info.AppID != 0 {
			gc = LookupGameByAppID(info.AppID)
		} else if info.Protocol == "minecraft" {
			gc = LookupGame("minecraft")
		}
	}

	if gc != nil && gc.DefaultQueryPort > gc.DefaultGamePort {
		offset := gc.DefaultQueryPort - gc.DefaultGamePort
		queriedPort := info.GamePort
		if queriedPort >= offset {
			info.GamePort = queriedPort - offset
			info.QueryPort = queriedPort
		}
	}

	if gc != nil {
		info.Game = gc.Name
	} else {
		info.Game = sanitize(info.Game)
	}
	sanitizeInfo(info)
}

var tagRegex = regexp.MustCompile(`<[^>]+>`)

func sanitize(s string) string {
	return strings.TrimSpace(tagRegex.ReplaceAllString(s, ""))
}

func sanitizeInfo(info *ServerInfo) {
	info.Name = sanitize(info.Name)
	info.Map = sanitize(info.Map)
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

func collectPorts(portRanges []PortRange) []uint16 {
	var ports []uint16

	if len(portRanges) > 0 {
		for _, pr := range portRanges {
			for port := uint32(pr.Start); port <= uint32(pr.End); port++ {
				ports = append(ports, uint16(port))
			}
		}
	} else {
		for _, g := range SupportedGames() {
			ports = append(ports, g.DefaultQueryPort, g.DefaultGamePort)
		}
	}

	return dedupPorts(ports...)
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
