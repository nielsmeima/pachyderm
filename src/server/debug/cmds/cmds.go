package cmds

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"golang.org/x/sync/errgroup"

	units "github.com/docker/go-units"
	"github.com/pachyderm/pachyderm/src/client"
	"github.com/pachyderm/pachyderm/src/server/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// Cmds returns a slice containing debug commands.
func Cmds(noMetrics *bool, noPortForwarding *bool) []*cobra.Command {
	var commands []*cobra.Command

	dump := &cobra.Command{
		Use:   "dump",
		Short: "Return a dump of running goroutines.",
		Long:  "Return a dump of running goroutines.",
		Run: cmdutil.RunFixedArgs(0, func(args []string) error {
			client, err := client.NewOnUserMachine(!*noMetrics, !*noPortForwarding, "debug-dump")
			if err != nil {
				return err
			}
			defer client.Close()
			return client.Dump(os.Stdout)
		}),
	}
	commands = append(commands, dump)

	var duration time.Duration
	profile := &cobra.Command{
		Use:   "profile <profile>",
		Short: "Return a profile from the server.",
		Long:  "Return a profile from the server.",
		Run: cmdutil.RunFixedArgs(1, func(args []string) error {
			client, err := client.NewOnUserMachine(!*noMetrics, !*noPortForwarding, "debug-dump")
			if err != nil {
				return err
			}
			defer client.Close()
			return client.Profile(args[0], duration, os.Stdout)
		}),
	}
	profile.Flags().DurationVarP(&duration, "duration", "d", time.Minute, "Duration to run a CPU profile for.")
	commands = append(commands, profile)

	binary := &cobra.Command{
		Use:   "binary",
		Short: "Return the binary the server is running.",
		Long:  "Return the binary the server is running.",
		Run: cmdutil.RunFixedArgs(0, func(args []string) error {
			client, err := client.NewOnUserMachine(!*noMetrics, !*noPortForwarding, "debug-dump")
			if err != nil {
				return err
			}
			defer client.Close()
			return client.Binary(os.Stdout)
		}),
	}
	commands = append(commands, binary)

	var profileFile string
	var binaryFile string
	pprof := &cobra.Command{
		Use:   "pprof <profile>",
		Short: "Analyze a profile of pachd in pprof.",
		Long:  "Analyze a profile of pachd in pprof.",
		Run: cmdutil.RunFixedArgs(1, func(args []string) error {
			client, err := client.NewOnUserMachine(!*noMetrics, !*noPortForwarding, "debug-dump")
			if err != nil {
				return err
			}
			defer client.Close()
			var eg errgroup.Group
			// Download the profile
			eg.Go(func() (retErr error) {
				if args[0] == "cpu" {
					fmt.Printf("Downloading cpu profile, this will take %s...", units.HumanDuration(duration))
				}
				f, err := os.Create(profileFile)
				if err != nil {
					return err
				}
				defer func() {
					if err := f.Close(); err != nil && retErr == nil {
						retErr = err
					}
				}()
				return client.Profile(args[0], duration, f)
			})
			// Download the binary
			eg.Go(func() (retErr error) {
				f, err := os.Create(binaryFile)
				if err != nil {
					return err
				}
				defer func() {
					if err := f.Close(); err != nil && retErr == nil {
						retErr = err
					}
				}()
				return client.Binary(f)
			})
			if err := eg.Wait(); err != nil {
				return err
			}
			cmd := exec.Command("go", "tool", "pprof", binaryFile, profileFile)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}),
	}
	pprof.Flags().StringVar(&profileFile, "profile-file", "profile", "File to write the profile to.")
	pprof.Flags().StringVar(&binaryFile, "binary-file", "binary", "File to write the binary to.")
	pprof.Flags().DurationVarP(&duration, "duration", "d", time.Minute, "Duration to run a CPU profile for.")
	commands = append(commands, pprof)

	debug := &cobra.Command{
		Use:   "debug",
		Short: "Debug commands for analyzing a running cluster.",
		Long:  "Debug commands for analyzing a running cluster.",
	}

	for _, cmd := range commands {
		debug.AddCommand(cmd)
	}

	return []*cobra.Command{debug}
}
