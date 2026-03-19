package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/0xkowalskidev/gjq"
)

var sanitizeRe = regexp.MustCompile(strings.Join([]string{
	`§[0-9a-fk-or]`,  // Minecraft color/formatting codes
	`\x1b\[[0-9;]*m`, // ANSI escape sequences
	`<[^>]+>`,         // HTML/Unity rich text tags
	`\^(X[0-9a-fA-F]{6}|.)`, // Quake-style color codes (^N and ^Xrrggbb)
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

// sanitizeInfo cleans display strings on a copy so library consumers get raw data.
func sanitizeInfo(info *gjq.ServerInfo) *gjq.ServerInfo {
	out := *info
	out.Name = sanitize(out.Name)
	out.Game = sanitize(out.Game)
	out.Map = sanitize(out.Map)
	out.GameMode = sanitize(out.GameMode)
	out.Version = sanitize(out.Version)
	out.Keywords = sanitize(out.Keywords)
	out.ServerType = sanitize(out.ServerType)
	out.Environment = sanitize(out.Environment)
	out.Visibility = sanitize(out.Visibility)
	if len(out.PlayerList) > 0 {
		out.PlayerList = make([]gjq.PlayerInfo, len(info.PlayerList))
		copy(out.PlayerList, info.PlayerList)
		for i := range out.PlayerList {
			out.PlayerList[i].Name = sanitize(out.PlayerList[i].Name)
		}
	}
	return &out
}

func printServerInfo(info *gjq.ServerInfo, asJSON bool) error {
	if asJSON {
		return printJSON(info)
	}
	printTable(sanitizeInfo(info))
	return nil
}

func printMultiServerInfo(servers []*gjq.ServerInfo, asJSON bool) error {
	if asJSON {
		return printJSON(servers)
	}
	for i, s := range servers {
		if i > 0 {
			fmt.Println()
		}
		printTable(sanitizeInfo(s))
	}
	return nil
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printTable(info *gjq.ServerInfo) {
	fmt.Printf("%-12s %s\n", "Name:", info.Name)
	fmt.Printf("%-12s %s\n", "Address:", info.Address)
	fmt.Printf("%-12s %d\n", "Game Port:", info.GamePort)
	fmt.Printf("%-12s %d\n", "Query Port:", info.QueryPort)
	if info.Game != "" {
		fmt.Printf("%-12s %s\n", "Game:", info.Game)
	}
	if info.Map != "" {
		fmt.Printf("%-12s %s\n", "Map:", info.Map)
	}
	if info.GameMode != "" {
		fmt.Printf("%-12s %s\n", "Game Mode:", info.GameMode)
	}
	fmt.Printf("%-12s %d / %d\n", "Players:", info.Players, info.MaxPlayers)

	if info.Bots > 0 {
		fmt.Printf("%-12s %d\n", "Bots:", info.Bots)
	}
	if info.ServerType != "" {
		fmt.Printf("%-12s %s\n", "Type:", info.ServerType)
	}
	if info.Environment != "" {
		fmt.Printf("%-12s %s\n", "OS:", info.Environment)
	}
	if info.Visibility != "" {
		fmt.Printf("%-12s %s\n", "Visibility:", info.Visibility)
	}
	if info.VAC {
		fmt.Printf("%-12s %s\n", "VAC:", "enabled")
	}
	if info.Version != "" {
		fmt.Printf("%-12s %s\n", "Version:", info.Version)
	}
	if info.Keywords != "" {
		fmt.Printf("%-12s %s\n", "Keywords:", info.Keywords)
	}
	if len(info.Mods) > 0 {
		fmt.Printf("%-12s %d mods\n", "Mods:", len(info.Mods))
	}
	fmt.Printf("%-12s %s\n", "Protocol:", info.Protocol)
	fmt.Printf("%-12s %s\n", "Ping:", info.Ping.Round(time.Millisecond).String())

	if len(info.PlayerList) > 0 {
		fmt.Printf("\n%-30s %-8s %s\n", "PLAYER", "SCORE", "DURATION")
		fmt.Printf("%-30s %-8s %s\n", "------", "-----", "--------")
		for _, p := range info.PlayerList {
			fmt.Printf("%-30s %-8d %s\n", p.Name, p.Score, p.Duration.Round(time.Second))
		}
	}
}
