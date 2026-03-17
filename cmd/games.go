package cmd

import (
	"fmt"
	"strings"

	"github.com/0xkowalskidev/gjq"
	"github.com/spf13/cobra"
)

func NewGamesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "games",
		Short: "List supported games",
		RunE: func(cmd *cobra.Command, args []string) error {
			games := gjq.SupportedGames()

			// Build combined game column: "slug (alias1, alias2)" or just "slug"
			gameLabels := make([]string, len(games))
			gameWidth := len("GAME")
			for i, g := range games {
				if len(g.Aliases) > 0 {
					gameLabels[i] = fmt.Sprintf("%s (%s)", g.Slug, strings.Join(g.Aliases, ", "))
				} else {
					gameLabels[i] = g.Slug
				}
				if len(gameLabels[i]) > gameWidth {
					gameWidth = len(gameLabels[i])
				}
			}
			gameWidth += 2

			fmtStr := fmt.Sprintf("%%-%ds %%-10s %%-12s %%-10s %%s\n", gameWidth)

			fmt.Printf(fmtStr, "GAME", "GAME PORT", "QUERY PORT", "PROTOCOL", "SUPPORTS")
			fmt.Printf(fmtStr, strings.Repeat("-", gameWidth-2), "---------", "----------", "--------", "--------")

			for i, g := range games {
				sup := "-"
				if len(g.Supports) > 0 {
					sup = strings.Join(g.Supports, ", ")
				}
				fmt.Printf(fmt.Sprintf("%%-%ds %%-10d %%-12d %%-10s %%s\n", gameWidth), gameLabels[i], g.DefaultGamePort, g.DefaultQueryPort, g.Protocol, sup)
				if g.Notes != "" {
					fmt.Printf("Notes: %s\n\n", g.Notes)
				}
			}

			return nil
		},
	}
}
