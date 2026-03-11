package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/0xkowalskidev/gsq"
	"github.com/spf13/cobra"
)

var (
	flagGame    string
	flagJSON    bool
	flagTimeout time.Duration
	flagDebug bool
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gsq [flags] host:port",
		Short: "Query game servers",
		Long:  "gsq queries game servers using various protocols (Source, Minecraft, etc.)",
		Args: cobra.MaximumNArgs(1),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			level := slog.LevelInfo
			if flagDebug || os.Getenv("DEBUG") == "1" {
				level = slog.LevelDebug
			}
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			host, portStr, err := net.SplitHostPort(args[0])
			if err != nil {
				// No port given — try to infer from --game
				if flagGame == "" {
					return fmt.Errorf("no port specified and no --game flag set — use host:port or --game to infer default port")
				}
				gc := gsq.LookupGame(flagGame)
				if gc == nil {
					return fmt.Errorf("unknown game %q — run 'gsq games' to see supported games", flagGame)
				}
				host = args[0]
				portStr = strconv.FormatUint(uint64(gc.DefaultGamePort), 10)
			}

			port, err := strconv.ParseUint(portStr, 10, 16)
			if err != nil {
				return fmt.Errorf("invalid port %q — must be a number between 1 and 65535", portStr)
			}

			ctx := context.Background()
			opts := gsq.QueryOptions{
				Game:    flagGame,
				Timeout: flagTimeout,
			}

			info, err := gsq.Query(ctx, host, uint16(port), opts)
			if err != nil {
				return err
			}

			return printServerInfo(info, flagJSON)
		},
	}

	rootCmd.Flags().StringVar(&flagGame, "game", "", "game slug to skip auto-detection (e.g. cs2, minecraft)")
	rootCmd.Flags().BoolVar(&flagJSON, "json", false, "output as JSON")
	rootCmd.Flags().DurationVar(&flagTimeout, "timeout", 5*time.Second, "query timeout")
	rootCmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "enable debug logging")

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(NewGamesCmd())
	rootCmd.AddCommand(NewScanCmd())

	return rootCmd
}
