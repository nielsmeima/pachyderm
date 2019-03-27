package cmd

import (
	"fmt"
	"strings"

	"github.com/pachyderm/pachyderm/src/server/pkg/cmdutil"

	"github.com/spf13/cobra"
)
func apply_v1_8_command_compat(rootCmd *cobra.Command) {
	var commands []*cobra.Command

	// Helper functions to avoid repetition
	findCommand := func (fullName string) *cobra.Command {
		cursor := rootCmd
		for _, name := range strings.SplitN(fullName, " ", -1) {
			var err error
			cursor, _, err = cursor.Find([]string{name})
			if err != nil {
				panic(fmt.Sprintf("Could not find '%s' command to apply v1.8 compatibility\n", fullName))
			}
		}
		return cursor
	}

	// These commands are backwards compatible aside from the reorganization, just
	// add new aliases.
	simpleCompat := map[string]string{
		// TODO this command used to be deprecated and was removed: "set branch": "set-branch"
		"create repo": "create-repo",
		"update repo": "update-repo",
		"inspect repo": "inspect-repo",
		"list repo": "list-repo",
		"delete repo": "delete-repo",
		"list commit": "list-commit",
		"list branch": "list-branch",
		"get object": "get-object",
		"get tag": "get-tag",
		"inspect job": "inspect-job",
		"delete job": "delete-job",
		"stop job": "stop-job",
		"restart datum": "restart-datum",
		"list datum": "list-datum",
		"inspect datum": "inspect-datum",
		"get logs": "get-logs",
		"create pipeline": "create-pipeline",
		"update pipeline": "update-pipeline",
		"inspect pipeline": "inspect-pipeline",
		"extract pipeline": "extract-pipeline",
		"edit pipeline": "edit-pipeline",
		"list pipeline": "list-pipeline",
		"delete pipeline": "delete-pipeline",
		"start pipeline": "start-pipeline",
		"stop pipeline": "stop-pipeline",
		"inspect cluster": "inspect-cluster",
		"debug dump": "debug-dump",
		"debug profile": "debug-profile",
		"debug binary": "debug-binary",
		"debug pprof": "debug-pprof",
		"delete all": "delete-all",
	}

	for newName, oldName := range simpleCompat {
		compatCmd := &cobra.Command{}
		*compatCmd = *findCommand(newName)

		useSplit := strings.SplitN(compatCmd.Use, " ", 1)
		if len(useSplit) == 2 {
			compatCmd.Use = fmt.Sprintf("%s %s", oldName, useSplit[1])
		} else {
			compatCmd.Use = oldName
		}

		commands = append(commands, compatCmd)
	}

	// Helper types for organizing more complicated command compatibility
	type RunFunc func(*cobra.Command, []string)
	type CompatChanges struct {
		Use string
		Example string
		Run func(RunFunc) RunFunc
	}

	// These helper functions will transform positional command-line args and
	// pass-through to the new implementations of the command so we can maintain
	// a single code path.
	transformRepoBranch := func(newRun RunFunc) func([]string) error {
		return func(args []string) error {
			var newArgs []string

			newRun(nil, newArgs)
			return nil
		}
	}

	transformRepoBranchFile := func(newRun RunFunc) func([]string) error {
		return func(args []string) error {
			var newArgs []string

			newRun(nil, newArgs)
			return nil
		}
	}

	transformRepoSlashBranch := func(newRun RunFunc) func([]string) error {
		return func(args []string) error {
			var newArgs []string

			for _, arg := range args {
				newArgs = append(newArgs, strings.Replace(arg, "/", "@", 1))
			}

			newRun(nil, newArgs)
			return nil
		}
	}

	// These commands require transforming their old parameter format to the new
	// format, as well as maintaining their old Use strings.
	complexCompat := map[string]CompatChanges{
		"start commit": {
			Use: "start-commit <repo> [<branch-or-commit>]",
			Example: `
# Start a new commit in repo "test" that's not on any branch
$ pachctl start-commit test

# Start a commit in repo "test" on branch "master"
$ pachctl start-commit test master

# Start a commit with "master" as the parent in repo "test", on a new branch "patch"; essentially a fork.
$ pachctl start-commit test patch -p master

# Start a commit with XXX as the parent in repo "test", not on any branch
$ pachctl start-commit test -p XXX
			`,
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunBoundedArgs(1, 2, transformRepoBranch(newRun))
			},
		},

		"finish commit": {
			Use: "finish-commit <repo> <branch-or-commit>",
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunFixedArgs(2, transformRepoBranch(newRun))
			},
		},

		"inspect commit": {
			Use: "inspect-commit <repo> <branch-or-commit>",
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunFixedArgs(2, transformRepoBranch(newRun))
			},
		},

		"subscribe commit": {
			Use: "subscribe-commit <repo> <branch>",
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunFixedArgs(2, transformRepoBranch(newRun))
			},
		},

		"delete commit": {
			Use: "delete-commit <repo> <commit>",
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunFixedArgs(2, transformRepoBranch(newRun))
			},
		},

		"delete branch": {
			Use: "delete-branch <repo> <branch>",
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunFixedArgs(2, transformRepoBranch(newRun))
			},
		},

		"flush job": {
			Use:     "flush-job <repo>/<commit> ...",
			Example: `
# return jobs caused by foo/XXX and bar/YYY
$ pachctl flush-job foo/XXX bar/YYY

# return jobs caused by foo/XXX leading to pipelines bar and baz
$ pachctl flush-job foo/XXX -p bar -p baz`,
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.Run(transformRepoSlashBranch(newRun))
			},
		},

		"flush commit": {
			Use:   "flush-commit <repo>/<commit> ...",
			Example: `
# return commits caused by foo/XXX and bar/YYY
$ pachctl flush-commit foo/XXX bar/YYY

# return commits caused by foo/XXX leading to repos bar and baz
$ pachctl flush-commit foo/XXX -r bar -r baz`,
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.Run(transformRepoSlashBranch(newRun))
			},
		},

		"put file": {
			Use:     "put-file repo-name branch [path/to/file/in/pfs]",
			Example: `
# Put data from stdin as repo/branch/path:
$ echo "data" | pachctl put-file repo branch path

# Put data from stdin as repo/branch/path and start / finish a new commit on the branch.
$ echo "data" | pachctl put-file -c repo branch path

# Put a file from the local filesystem as repo/branch/path:
$ pachctl put-file repo branch path -f file

# Put a file from the local filesystem as repo/branch/file:
$ pachctl put-file repo branch -f file

# Put the contents of a directory as repo/branch/path/dir/file:
$ pachctl put-file -r repo branch path -f dir

# Put the contents of a directory as repo/branch/dir/file:
$ pachctl put-file -r repo branch -f dir

# Put the contents of a directory as repo/branch/file, i.e. put files at the top level:
$ pachctl put-file -r repo branch / -f dir

# Put the data from a URL as repo/branch/path:
$ pachctl put-file repo branch path -f http://host/path

# Put the data from a URL as repo/branch/path:
$ pachctl put-file repo branch -f http://host/path

# Put the data from an S3 bucket as repo/branch/s3_object:
$ pachctl put-file repo branch -r -f s3://my_bucket

# Put several files or URLs that are listed in file.
# Files and URLs should be newline delimited.
$ pachctl put-file repo branch -i file

# Put several files or URLs that are listed at URL.
# NOTE this URL can reference local files, so it could cause you to put sensitive
# files into your Pachyderm cluster.
$ pachctl put-file repo branch -i http://host/path`,
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunBoundedArgs(2, 3, transformRepoBranchFile(newRun))
			},
		},

		"get file": {
			Use:     "get-file <repo> <commit> <path/to/file>",
			Example: `
# get file "XXX" on branch "master" in repo "foo"
$ pachctl get-file foo master XXX

# get file "XXX" in the parent of the current head of branch "master"
# in repo "foo"
$ pachctl get-file foo master^ XXX

# get file "XXX" in the grandparent of the current head of branch "master"
# in repo "foo"
$ pachctl get-file foo master^2 XXX`,
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunFixedArgs(3, transformRepoBranchFile(newRun))
			},
		},

		"inspect file": {
			Use:   "inspect-file <repo> <commit> <path/to/file>",
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunFixedArgs(3, transformRepoBranchFile(newRun))
			},
		},

		"list file": {
			Use:   "list-file repo-name commit-id path/to/dir",
			Example: `
# list top-level files on branch "master" in repo "foo"
$ pachctl list-file foo master

# list files under directory "dir" on branch "master" in repo "foo"
$ pachctl list-file foo master dir

# list top-level files in the parent commit of the current head of "master"
# in repo "foo"
$ pachctl list-file foo master^

# list top-level files in the grandparent of the current head of "master"
# in repo "foo"
$ pachctl list-file foo master^2

# list the last n versions of top-level files on branch "master" in repo "foo"
$ pachctl list-file foo master --history n

# list all versions of top-level files on branch "master" in repo "foo"
$ pachctl list-file foo master --history -1`,
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunBoundedArgs(2, 3, transformRepoBranchFile(newRun))
			},
		},

		"glob file": {
			Use:     "glob-file <repo> <commit> <pattern>",
			Example: `
# Return files in repo "foo" on branch "master" that start
# with the character "A".  Note how the double quotation marks around "A*" are
# necessary because otherwise your shell might interpret the "*".
$ pachctl glob-file foo master "A*"

# Return files in repo "foo" on branch "master" under directory "data".
$ pachctl glob-file foo master "data/*"`,
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunFixedArgs(3, transformRepoBranchFile(newRun))
			},
		},

		"delete file": {
			Use: "delete-file <repo> <commit> <path/to/file>",
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunFixedArgs(3, transformRepoBranchFile(newRun))
			},
		},

		"copy file": {
			Use: "copy-file <src-repo> <src-commit> <src-path> <dst-repo> <dst-commit> <dst-path>",
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunFixedArgs(6, transformRepoBranchFile(newRun))
			},
		},

		"diff file": {
			Use:     "diff-file <new-repo> <new-commit> <new-path> [<old-repo> <old-commit> <old-path>]",
			Example: `
# Return the diff between foo master path and its parent.
$ pachctl diff-file foo master path

# Return the diff between foo master path1 and bar master path2.
$ pachctl diff-file foo master path1 bar master path2`,
			Run: func (newRun RunFunc) RunFunc {
				return cmdutil.RunBoundedArgs(3, 6, transformRepoBranchFile(newRun))
			},
		},
	}

	for newName, changes := range complexCompat {
		newCmd := findCommand(newName)
		oldCmd := &cobra.Command{}
		*oldCmd = *newCmd
		oldCmd.Use = changes.Use
		oldCmd.Example = changes.Example
		oldCmd.Run = changes.Run(newCmd.Run)
		commands = append(commands, oldCmd)
	}

	oldParseBranches := func()

	// create-branch has affected flags, so just duplicate the command entirely
	var branchProvenance cmdutil.RepeatedStringArg
	var head string
	newCreateBranch := findCommand("create branch")
	oldCreateBranch := &cobra.Command{
		Use:   "create-branch <repo> <branch>",
		Short: newCreateBranch.Short
		Long:  newCreateBranch.Long
		Run: cmdutil.RunFixedArgs(2, func(args []string) error {
			client, err := client.NewOnUserMachine(!*noMetrics, !*noPortForwarding, "user")
			if err != nil {
				return err
			}
			defer client.Close()
			provenance, err := oldParseBranches(branchProvenance)
			if err != nil {
				return err
			}
			return client.CreateBranch(args[0], args[1], head, provenance)
		}),
	}
	oldCreateBranch.Flags().VarP(&branchProvenance, "provenance", "p", "The provenance for the branch.")
	oldCreateBranch.Flags().StringVarP(&head, "head", "", "", "The head of the newly created branch.")

	// list-job has affected flags, so just duplicate the command entirely
	var raw bool
	var fullTimestamps bool
	var pipelineName string
	var outputCommitStr string
	var inputCommitStrs []string
	newListJob := findCommand("list job")
	oldListJob := &cobra.Command{
		Use:     "list-job",
		Short:   newListJob.Short
		Long:    newListJob.Long
		Example: `
# return all jobs
$ pachctl list-job

# return all jobs in pipeline foo
$ pachctl list-job -p foo

# return all jobs whose input commits include foo/XXX and bar/YYY
$ pachctl list-job foo/XXX bar/YYY

# return all jobs in pipeline foo and whose input commits include bar/YYY
$ pachctl list-job -p foo bar/YYY`,
		Run:     cmdutil.RunFixedArgs(0, func(args []string) error {
			client, err := pachdclient.NewOnUserMachine(!*noMetrics, !*noPortForwarding, "user")
			if err != nil {
				return err
			}
			defer client.Close()

			commits, err := oldParseCommits(inputCommitStrs)
			if err != nil {
				return err
			}

			var outputCommit *pfs.Commit
			if outputCommitStr != "" {
				outputCommits, err := oldParseCommits([]string{outputCommitStr})
				if err != nil {
					return err
				}
				if len(outputCommits) == 1 {
					outputCommit = outputCommits[0]
				}
			}

			if raw {
				return client.ListJobF(pipelineName, commits, outputCommit, func(ji *ppsclient.JobInfo) error {
					if err := marshaller.Marshal(os.Stdout, ji); err != nil {
						return err
					}
					return nil
				})
			}
			writer := tabwriter.NewWriter(os.Stdout, pretty.JobHeader)
			if err := client.ListJobF(pipelineName, commits, outputCommit, func(ji *ppsclient.JobInfo) error {
				pretty.PrintJobInfo(writer, ji, fullTimestamps)
				return nil
			}); err != nil {
				return err
			}
			return writer.Flush()
		}),
	}
	listJob.Flags().StringVarP(&pipelineName, "pipeline", "p", "", "Limit to jobs made by pipeline.")
	listJob.Flags().StringVarP(&outputCommitStr, "output", "o", "", "List jobs with a specific output commit.")
	listJob.Flags().StringSliceVarP(&inputCommitStrs, "input", "i", []string{}, "List jobs with a specific set of input commits.")
	listJob.Flags().BoolVar(&fullTimestamps, "full-timestamps", false, "Return absolute timestamps (as opposed to the default, relative timestamps).")
	listJob.Flags().BoolVar(&raw, "raw", false, "disable pretty printing, print raw json")

	// Apply the 'Hidden' attribute to all these commands so they don't pollute help
	for _, cmd := range commands {
		cmd.Hidden = true
	}

	rootCmd.AddCommand(commands...)
}
