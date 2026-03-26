package cmd

import (
	"fmt"
	"strings"

	"github.com/warsmite/gjq"
	"github.com/spf13/cobra"
)

func NewGamesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "games",
		Short: "List supported games",
		RunE: func(cmd *cobra.Command, args []string) error {
			games := gjq.Registry.WithQuery()

			if flagJSON {
				return printJSON(games)
			}

			// Build combined game column: "id (alias1, alias2)" or just "id"
			gameLabels := make([]string, len(games))
			gameWidth := len("GAME")
			for i, g := range games {
				if len(g.Aliases) > 0 {
					gameLabels[i] = fmt.Sprintf("%s (%s)", g.ID, strings.Join(g.Aliases, ", "))
				} else {
					gameLabels[i] = g.ID
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
				if len(g.Query.Supports) > 0 {
					sup = strings.Join(g.Query.Supports, ", ")
				}
				fmt.Printf(fmt.Sprintf("%%-%ds %%-10d %%-12d %%-10s %%s\n", gameWidth), gameLabels[i], g.GamePort(), g.QueryPort(), g.Query.Protocol, sup)
				if g.Query.Notes != "" {
					fmt.Printf("Notes: %s\n\n", g.Query.Notes)
				}
			}

			return nil
		},
	}

	return cmd
}
