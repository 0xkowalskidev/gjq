package cmd

import (
	"fmt"
	"strings"

	"github.com/0xkowalskidev/gsq"
	"github.com/spf13/cobra"
)

func NewGamesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "games",
		Short: "List supported games",
		RunE: func(cmd *cobra.Command, args []string) error {
			games := gsq.SupportedGames()

			// Find max slug width for alignment
			slugWidth := len("SLUG")
			for _, g := range games {
				if len(g.Slug) > slugWidth {
					slugWidth = len(g.Slug)
				}
			}
			slugWidth += 2

			fmtStr := fmt.Sprintf("%%-%ds %%-10s %%-12s %%-15s %%s\n", slugWidth)

			fmt.Printf(fmtStr, "SLUG", "GAME PORT", "QUERY PORT", "ALIASES", "PROTOCOL")
			fmt.Printf(fmtStr, strings.Repeat("-", slugWidth-2), "---------", "----------", "-------", "--------")

			for _, g := range games {
				aliases := "-"
				if len(g.Aliases) > 0 {
					aliases = strings.Join(g.Aliases, ", ")
				}
				fmt.Printf(fmt.Sprintf("%%-%ds %%-10d %%-12d %%-15s %%s\n", slugWidth), g.Slug, g.DefaultGamePort, g.DefaultQueryPort, aliases, g.Protocol)
			}

			return nil
		},
	}
}
