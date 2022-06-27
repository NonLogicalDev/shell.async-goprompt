package main

import (
	"EXP/pkg/shellout"
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
	"sync"
	"time"
)

var bgctx = context.Background()

var (
	cmd = &cobra.Command{
		Use: "goprompt",
	}
	flgCmdStatus = cmd.PersistentFlags().Int(
		"cmd-status", 0,
		"cmd status of previous command",
	)
	flgPreexecTS = cmd.PersistentFlags().Int(
		"preexec-ts", 0,
		"pre-execution timestamp to gauge how log execution took",
	)
)

func init() {
	cmd.RunE = cmdExec
}

// PROMPT PARTS:
// (exit-status: if > 0)
// (parent-process)
// (hostname: if remote connection)
// (current-dir-path)
// (vsc-information)
// (timestamp)

func cmdExec(cmd *cobra.Command, args []string) error {
	if *flgCmdStatus != 0 {
		printPart("st", fmt.Sprintf("%#v", *flgCmdStatus))
	}

	wg := new(WaitGroupDispatcher)
	defer wg.Wait()

	wg.Dispatch(func() {
		homeDir := os.Getenv("HOME")

		if wd, err := os.Getwd(); err == nil {
			wdh := strings.Replace(wd, homeDir, "~", 1)

			printPart("wd_full", wdh)
			printPart("wd", trimPath(wdh))
		}

		nowTS := time.Now()
		printPart("ts", nowTS.Format("15:04:05 01/02/06"))

		if *flgPreexecTS != 0 {
			cmdTS := time.Unix(int64(*flgPreexecTS), 0)

			diff := nowTS.Sub(cmdTS)
			printPart("ds", diff.Round(time.Second))
		}
	})

	//wg.Dispatch(func() {
	//	out, err := stringExec("git", "config", "--list")
	//	printPart("debug_o", js(out))
	//	if err != nil {
	//		printPart("debug_e", js(err.Error()))
	//	}
	//})

	wg.Dispatch(func() {
		cwg := new(WaitGroupDispatcher)
		defer cwg.Wait()

		if _, err := stringExec("git", "rev-parse", "--show-toplevel"); err == nil {
			printPart("vcs", "git")
		} else {
			return
		}

		cwg.Dispatch(func() {
			if branch, err := stringExec("git", "branch", "--show-current"); err == nil {
				printPart("vcs_br", trim(string(branch)))
			}
		})

		cwg.Dispatch(func() {
			if status, err := stringExec("git", "status", "--porcelain"); err == nil {
				if len(status) > 0 {
					printPart("vcs_dirty", 1)
					printPart("vcs_dirty_st", js(status))
				} else {
					printPart("vsc_dirty", 0)
				}
			}
		})
	})

	wg.Dispatch(func() {
		cwg := new(WaitGroupDispatcher)
		defer cwg.Wait()

		var stgPatchTop string
		var err error

		if stgPatchTop, err = stringExec("stg", "top"); err == nil {
			printPart("stg", "1")
			printPart("stg_top", stgPatchTop)
		} else {
			return
		}

		cwg.Dispatch(func() {
			gitSHA, _ := stringExec("stg", "id")
			stgSHA, _ := stringExec("stg", "id", stgPatchTop)

			if gitSHA != stgSHA {
				printPart("stg_dirty", 1)
			} else {
				printPart("stg_dirty", 0)
			}
		})

		cwg.Dispatch(func() {
			if stgPatchLen, err := stringExec("stg", "series", "-c"); err == nil {
				printPart("stg_qlen", stgPatchLen)
			}
		})

		cwg.Dispatch(func() {
			if stgPatchPos, err := stringExec("stg", "series", "-cA"); err == nil {
				printPart("stg_qpos", stgPatchPos)
			}
		})
	})

	return nil
}

func printPart(name string, value interface{}) {
	if _, err := os.Stdout.Stat(); err != nil {
		os.Exit(1)
	}
	fmt.Printf("%s\t%v\n", name, value)
}

//------------------------------------------------------------------------------

func main() {
	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		panic(err)
	}
}

//------------------------------------------------------------------------------

type WaitGroupDispatcher struct {
	wg sync.WaitGroup
}

func (d *WaitGroupDispatcher) Dispatch(fn func()) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		fn()
	}()
}

func (d *WaitGroupDispatcher) Wait() {
	d.wg.Wait()
}

func stringExec(path string, args ...string) (string, error) {
	out, err := shellout.New(bgctx,
		shellout.Args(path, args...),
		shellout.EnvSet(map[string]string{
			"GIT_OPTIONAL_LOCKS": "0",
		}),
	).RunString()
	return trim(out), err
}

func js(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func trimPath(s string) string {
	var out []string

	parts := strings.Split(s, "/")
	for i, part := range parts {
		if i == len(parts)-1 {
			out = append(out, part)
		} else {
			out = append(out, part[0:intMin(len(part), 1)])
		}
	}

	return strings.Join(out, "/")
}

func intMax(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func intMin(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func trim(s string) string {
	return strings.Trim(s, "\n\t ")
}
