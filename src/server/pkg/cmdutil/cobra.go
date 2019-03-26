package cmdutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/pachyderm/pachyderm/src/client/pfs"
	"github.com/spf13/cobra"
)

// RunFixedArgs wraps a function in a function
// that checks its exact argument count.
func RunFixedArgs(numArgs int, run func([]string) error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if len(args) != numArgs {
			fmt.Printf("expected %d arguments, got %d\n\n", numArgs, len(args))
			cmd.Usage()
		} else {
			if err := run(args); err != nil {
				ErrorAndExit("%v", err)
			}
		}
	}
}

// RunBoundedArgs wraps a function in a function
// that checks its argument count is within a range.
func RunBoundedArgs(min int, max int, run func([]string) error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if len(args) < min || len(args) > max {
			fmt.Printf("expected %d to %d arguments, got %d\n\n", min, max, len(args))
			cmd.Usage()
		} else {
			if err := run(args); err != nil {
				ErrorAndExit("%v", err)
			}
		}
	}
}

// Run makes a new cobra run function that wraps the given function.
func Run(run func(args []string) error) func(*cobra.Command, []string) {
	return func(_ *cobra.Command, args []string) {
		if err := run(args); err != nil {
			ErrorAndExit(err.Error())
		}
	}
}

// ErrorAndExit errors with the given format and args, and then exits.
func ErrorAndExit(format string, args ...interface{}) {
	if errString := strings.TrimSpace(fmt.Sprintf(format, args...)); errString != "" {
		fmt.Fprintf(os.Stderr, "%s\n", errString)
	}
	os.Exit(1)
}

// ParseCommit takes an argument of the form "repo[@branch-or-commit]" and
// returns the corresponding *pfs.Commit.
func ParseCommit(arg string) (*pfs.Commit, error) {
	parts := strings.SplitN(arg, "@", 2)
	if parts[0] == "" {
		return nil, fmt.Errorf("invalid format \"%s\": repo cannot be empty", arg)
	}
	commit := &pfs.Commit{
		Repo: &pfs.Repo{
			Name: parts[0],
		},
		ID: "",
	}
	if len(parts) == 2 {
		commit.ID = parts[1]
	}
	return commit, nil
}

// ParseCommits converts all arguments to *pfs.Commit structs using the
// semantics of ParseCommit
func ParseCommits(args []string) ([]*pfs.Commit, error) {
	var results []*pfs.Commit
	for _, arg := range args {
		commit, err := ParseCommit(arg)
		if err != nil {
			return nil, err
		}
		results = append(results, commit)
	}
	return results, nil
}

// ParseBranch takes an argument of the form "repo[@branch]" and
// returns the corresponding *pfs.Branch.  This uses ParseBranch under the hood
// because a branch name is usually interchangeable with a commit-id.
func ParseBranch(arg string) (*pfs.Branch, error) {
	commit, err := ParseCommit(arg)
	if err != nil {
		return nil, err
	}
	return &pfs.Branch{Repo: commit.Repo, Name: commit.ID}, nil
}

// ParseBranches converts all arguments to *pfs.Commit structs using the
// semantics of ParseBranch
func ParseBranches(args []string) ([]*pfs.Branch, error) {
	var results []*pfs.Branch
	for _, arg := range args {
		branch, err := ParseBranch(arg)
		if err != nil {
			return nil, err
		}
		results = append(results, branch)
	}
	return results, nil
}

// ParseFile takes an argument of the form "repo[@branch-or-commit[:path]]", and
// returns the corresponding *pfs.File.
func ParseFile(arg string) (*pfs.File, error) {
	repoAndRest := strings.SplitN(arg, "@", 2)
	if repoAndRest[0] == "" {
		return nil, fmt.Errorf("invalid format \"%s\": repo cannot be empty", arg)
	}
	file := &pfs.File{
		Commit: &pfs.Commit{
			Repo: &pfs.Repo{
				Name: repoAndRest[0],
			},
			ID: "",
		},
		Path: "",
	}
	if len(repoAndRest) > 1 {
		commitAndPath := strings.SplitN(repoAndRest[1], ":", 2)
		if commitAndPath[0] == "" {
			return nil, fmt.Errorf("invalid format \"%s\": commit cannot be empty", arg)
		}
		file.Commit.ID = commitAndPath[0]
		if len(commitAndPath) > 1 {
			file.Path = commitAndPath[1]
		}
	}
	return file, nil
}

// ParseFiles converts all arguments to *pfs.Commit structs using the
// semantics of ParseFile
func ParseFiles(args []string) ([]*pfs.File, error) {
	var results []*pfs.File
	for _, arg := range args {
		commit, err := ParseFile(arg)
		if err != nil {
			return nil, err
		}
		results = append(results, commit)
	}
	return results, nil
}

// RepeatedStringArg is an alias for []string
type RepeatedStringArg []string

func (r *RepeatedStringArg) String() string {
	result := "["
	for i, s := range *r {
		if i != 0 {
			result += ", "
		}
		result += s
	}
	return result + "]"
}

// Set adds a string to r
func (r *RepeatedStringArg) Set(s string) error {
	*r = append(*r, s)
	return nil
}

// Type returns the string representation of the type of r
func (r *RepeatedStringArg) Type() string {
	return "[]string"
}

// SetDocsUsage sets the usage string for a docs-style command.  Docs commands
// have no functionality except to output some docs and related commands, and
// should not specify a 'Run' attribute.
func SetDocsUsage(command *cobra.Command, subcommands []*cobra.Command) {
    command.SetUsageTemplate(`Usage:
  pachctl [command]{{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}
{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}
`)

    command.SetHelpTemplate(`{{or .Long .Short}}
{{.UsageString}}`)

    // This song-and-dance is so that we can render the related commands without
    // actually having them usable as subcommands of the docs command.
    // That is, we don't want `pachctl job list-job` to work, it should just
    // be `pachctl list-job`.  Therefore, we lazily add/remove the subcommands
    // only when we try to render usage for the docs command.
    originalUsage := command.UsageFunc()
    command.SetUsageFunc(func (c *cobra.Command) error {
        newUsage := command.UsageFunc()
        command.SetUsageFunc(originalUsage)
        defer command.SetUsageFunc(newUsage)

        command.AddCommand(subcommands...)
        defer command.RemoveCommand(subcommands...)

        command.Usage()
        return nil
    })
}

// Generates one or many nested command trees for each invocation listed in
// 'invocations', which should be space-delimited as on the command-line.  The
// 'Use' field of 'cmd' should not include the name of the command as that will
// be filled in based on each invocation.  These commands can later be merged
// into the final Command tree using 'MergeCommands' below.
func CreateAliases(cmd *cobra.Command, invocations []string) []*cobra.Command {
	var aliases []*cobra.Command

	// Create logical commands for each substring in each invocation
	for _, invocation := range invocations {
		var root, prev *cobra.Command
		args := strings.Split(invocation, " ")

		for i, arg := range args {
			cur := &cobra.Command{}

			// The leaf command node should include the usage from the given cmd,
			// while logical nodes just need one piece of the invocation.
			if i == len(args) - 1 {
				*cur = *cmd
				if cmd.Use == "" {
					cur.Use = arg
				} else {
					cur.Use = fmt.Sprintf("%s %s", arg, cmd.Use)
				}
			} else {
				cur.Use = arg
			}

			if root == nil {
				root = cur
			} else if prev != nil {
				prev.AddCommand(cur)
			}
			prev = cur
		}

		aliases = append(aliases, root)
	}

	return aliases
}

// This merges several command aliases (generated by 'CreateAliases' above) into
// a single coherent cobra command tree (with root command 'root').  Because
// 'CreateAliases' generates empty commands to preserve
func MergeCommands(root *cobra.Command, children []*cobra.Command) {
	// Move over any commands without subcommands, save the rest into a new slice
	var nested []*cobra.Command
	for _, cmd := range children {
    if cmd.HasSubCommands() {
			nested = append(nested, cmd)
		} else {
			root.AddCommand(cmd)
		}
	}

	for _, cmd := range nested {
		parent, _, err := root.Find([]string{cmd.Name()})
		if err != nil {
			root.AddCommand(cmd)
		} else {
			MergeCommands(parent, cmd.Commands())
		}
	}
}
