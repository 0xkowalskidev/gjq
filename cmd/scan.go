package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/0xkowalskidev/gsq"
	"github.com/spf13/cobra"
)

var (
	flagPorts       string
	flagScanTimeout time.Duration
)

func NewScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan [flags] host",
		Short: "Scan a host for game servers",
		Long:  "Probes all known game server ports (or a custom range) to find running servers.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			host := args[0]
			ctx := context.Background()

			opts := gsq.DiscoverOptions{
				Timeout: flagScanTimeout,
			}

			if flagPorts != "" {
				portRange, err := parsePortRange(flagPorts)
				if err != nil {
					return err
				}
				opts.PortRanges = []gsq.PortRange{portRange}
			}

			servers, err := gsq.Discover(ctx, host, opts)
			if err != nil {
				return err
			}

			if len(servers) == 0 {
				fmt.Println("No game servers found.")
				return nil
			}

			return printMultiServerInfo(servers, flagJSON)
		},
	}

	cmd.Flags().StringVar(&flagPorts, "ports", "", "port range to scan (e.g. 25000-26000)")
	cmd.Flags().DurationVar(&flagScanTimeout, "timeout", 1*time.Second, "per-probe timeout")

	return cmd
}

func parsePortRange(s string) (gsq.PortRange, error) {
	var start, end uint16
	n, err := fmt.Sscanf(s, "%d-%d", &start, &end)
	if err != nil || n != 2 {
		return gsq.PortRange{}, fmt.Errorf("invalid port range %q — expected format: start-end (e.g. 25000-26000)", s)
	}
	if start > end {
		return gsq.PortRange{}, fmt.Errorf("invalid port range: start (%d) must be <= end (%d)", start, end)
	}
	return gsq.PortRange{Start: start, End: end}, nil
}
