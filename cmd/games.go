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

			// Build combined game column: "slug (alias1, alias2)" or just "slug"
			gameLabels := make([]string, len(games))
			gameWidth := len("GAME")
			supWidth := len("SUPPORTS")
			for i, g := range games {
				if len(g.Aliases) > 0 {
					gameLabels[i] = fmt.Sprintf("%s (%s)", g.Slug, strings.Join(g.Aliases, ", "))
				} else {
					gameLabels[i] = g.Slug
				}
				if len(gameLabels[i]) > gameWidth {
					gameWidth = len(gameLabels[i])
				}
				s := strings.Join(g.Supports, ", ")
				if len(s) > supWidth {
					supWidth = len(s)
				}
			}
			gameWidth += 2
			supWidth += 2

			fmtStr := fmt.Sprintf("%%-%ds %%-10s %%-12s %%-10s %%-%ds %%s\n", gameWidth, supWidth)

			fmt.Printf(fmtStr, "GAME", "GAME PORT", "QUERY PORT", "PROTOCOL", "SUPPORTS", "NOTES")
			fmt.Printf(fmtStr, strings.Repeat("-", gameWidth-2), "---------", "----------", "--------", strings.Repeat("-", supWidth-2), "-----")

			for i, g := range games {
				sup := "-"
				if len(g.Supports) > 0 {
					sup = strings.Join(g.Supports, ", ")
				}
				notes := "-"
				if g.Notes != "" {
					notes = g.Notes
				}
				fmt.Printf(fmt.Sprintf("%%-%ds %%-10d %%-12d %%-10s %%-%ds %%s\n", gameWidth, supWidth), gameLabels[i], g.DefaultGamePort, g.DefaultQueryPort, g.Protocol, sup, notes)
			}

			return nil
		},
	}
}
