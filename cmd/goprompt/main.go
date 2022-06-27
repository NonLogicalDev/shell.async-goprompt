package main

import (
	"context"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"strings"
	"sync"
)

type CobraCallbackE func(cmd *cobra.Command, args []string) error

var (
	cmd = &cobra.Command{
		Use:               "goprompt",
		PersistentPreRunE: bindEnvironmentFlags("GOPROMPT"),
	}
	cmdQuery = &cobra.Command{
		Use:  "query",
		RunE: cmdQueryExec,
	}

	cmdQueryStatus = cmd.PersistentFlags().Int(
		"cmd-status", 0,
		"cmd status of previous command",
	)
	cmdQueryPreexecTS = cmd.PersistentFlags().String(
		"preexec-ts", "0",
		"pre-execution timestamp to gauge how log execution took",
	)
)

func init() {
	cmd.AddCommand(cmdQuery)

}

func bindEnvironmentFlags(prefix string) CobraCallbackE {
	return func(cmd *cobra.Command, args []string) (outErr error) {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if !f.Changed {
				envKey := prefix + "_" + strings.ReplaceAll(f.Name, "-", "_")
				if value, ok := os.LookupEnv(strings.ToUpper(envKey)); ok {
					if err := cmd.Flags().Set(f.Name, value); err != nil {
						outErr = err
						return
					}
				}
			}
		})
		return nil
	}
}

func cmdQueryExec(cmd *cobra.Command, args []string) error {
	if *cmdQueryStatus != 0 {
		printPart("st", fmt.Sprintf("%#v", *cmdQueryStatus))
	}

	wg := new(sync.WaitGroup)

	wg.Add(1)
	go func() {
		defer wg.Done()

		if wd, err := os.Getwd(); err == nil {
			printPart("wd", trimPathLast(wd, 2))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if branch, err := sh.Command("git", "branch", "--show-current").Output(); err == nil {
			printPart("git_br", trim(string(branch)))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if status, err := sh.Command("git", "status", "--porcelain").Output(); err == nil {
			if len(status) > 0 {
				printPart("git_st", "dirty")
			} else {
				printPart("git_st", "clean")
			}
		}
	}()

	wg.Wait()
	return nil
}

func trimPathLast(s string, n int) string {
	return s
}

func intMax(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func trim(s string) string {
	return strings.Trim(s, "\n\t ")
}

func printPart(name string, value interface{}) {
	if _, err := os.Stdout.Stat(); err != nil {
		os.Exit(1)
	}
	fmt.Printf("%s\t%v\n", name, value)
}

// PROMPT PARTS:
// (exit-status: if > 0)
// (parent-process)
// (hostname: if remote connection)
// (current-dir-path)
// (vsc-information)
// (timestamp)

func main() {
	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		panic(err)
	}
}
