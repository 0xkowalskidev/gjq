package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/0xkowalskidev/gsq"
)

func printServerInfo(info *gsq.ServerInfo, asJSON bool) error {
	if asJSON {
		return printJSON(info)
	}
	printTable(info)
	return nil
}

func printMultiServerInfo(servers []*gsq.ServerInfo, asJSON bool) error {
	if asJSON {
		return printJSON(servers)
	}
	for i, s := range servers {
		if i > 0 {
			fmt.Println()
		}
		printTable(s)
	}
	return nil
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printTable(info *gsq.ServerInfo) {
	fmt.Printf("%-12s %s\n", "Name:", info.Name)
	fmt.Printf("%-12s %s\n", "Address:", info.Address)
	fmt.Printf("%-12s %d\n", "Game Port:", info.GamePort)
	fmt.Printf("%-12s %d\n", "Query Port:", info.QueryPort)
	fmt.Printf("%-12s %s\n", "Game:", info.Game)
	fmt.Printf("%-12s %s\n", "Map:", info.Map)
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
