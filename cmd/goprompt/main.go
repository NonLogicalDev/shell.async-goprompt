package main

import (
	"EXP/pkg/shellout"
	"context"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/fatih/color"
)

var bgctx = context.Background()

var (
	cmd = &cobra.Command{
		Use: "goprompt",
	}

	cmdQuery = &cobra.Command{
		Use:   "query",
		Short: "start the query that pulls data for the prompt",
	}
	flgQCmdStatus = cmdQuery.PersistentFlags().Int(
		"cmd-status", 0,
		"cmd status of previous command",
	)
	flgQPreexecTS = cmdQuery.PersistentFlags().Int(
		"preexec-ts", 0,
		"pre-execution timestamp to gauge how log execution took",
	)

	cmdRender = &cobra.Command{
		Use:   "render",
		Short: "render the prompt based on the results of query",
	}
)

func init() {
	cmdQuery.RunE = cmdQueryRun
	cmd.AddCommand(cmdQuery)

	cmdRender.RunE = cmdRenderRun
	cmd.AddCommand(cmdRender)
}

func cmdRenderRun(cmd *cobra.Command, args []string) error {
	if _, err := os.Stdin.Stat(); err != nil {
		fmt.Printf("%#v", err)
	}

	out, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(out), "\n")
	p := make(map[string]string)
	for _, line := range lines {
		key, value, ok := strings.Cut(line, "\t")
		if ok {
			p[key] = value
		}
	}

	var partsTop []string
	if p["vcs"] == "git" {
		gitMark := color.GreenString("git")

		dirtyMarks := ""
		if p["vcs_dirty"] != "" && p["vcs_dirty"] != "0" {
			dirtyMarks = ":&"
			gitMark = color.RedString("git")
		}

		distanceMarks := ""
		distanceAhead := strInt(p["vcs_log_ahead"])
		distanceBehind := strInt(p["vcs_log_ahead"])
		if distanceAhead > 0 || distanceBehind > 0 {
			distanceMarks = fmt.Sprintf(":[+%v:-%v]", distanceAhead, distanceBehind)
		}

		partsTop = append(partsTop, fmt.Sprintf("{%v:%v%v%v}", gitMark, p["vcs_br"], dirtyMarks, distanceMarks))
	}

	fmt.Println("::", strings.Join(partsTop, " "))
	fmt.Printf(">")
	return nil
}

// PROMPT PARTS:
// (exit-status: if > 0)
// (parent-process)
// (hostname: if remote connection)
// (current-dir-path)
// (vsc-information)
// (timestamp)

func cmdQueryRun(cmd *cobra.Command, args []string) error {
	if *flgQCmdStatus != 0 {
		printPart("st", fmt.Sprintf("%#v", *flgQCmdStatus))
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

		if *flgQPreexecTS != 0 {
			cmdTS := time.Unix(int64(*flgQPreexecTS), 0)

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
				branch = trim(branch)
				if len(branch) > 0 {
					printPart("vcs_br", trim(branch))
					return
				}
			}

			if branch, err := stringExec("git", "name-rev", "--name-only", "HEAD"); err == nil {
				branch = trim(branch)
				if len(branch) > 0 {
					printPart("vcs_br", trim(branch))
					return
				}
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

		cwg.Dispatch(func() {
			if status, err := stringExec("git", "rev-list", "--left-right", "--count", "HEAD...@{u}"); err == nil {
				parts := strings.SplitN(status, "\t", 2)
				if len(parts) < 2 {
					parts = []string{"0", "0"}
				}

				printPart("vcs_log_ahead", parts[0])
				printPart("vcs_log_behind", parts[1])
			}
		})
	})

	wg.Dispatch(func() {
		var err error

		cwg := new(WaitGroupDispatcher)
		defer cwg.Wait()

		var stgSeriesLen string
		if stgSeriesLen, err = stringExec("stg", "series", "-c"); err == nil {
			printPart("stg", "1")
			printPart("stg_qlen", stgSeriesLen)
		}

		cwg.Dispatch(func() {
			if stgSeriesPos, err := stringExec("stg", "series", "-cA"); err == nil {
				printPart("stg_qpos", stgSeriesPos)
			}
		})

		var stgPatchTop string
		if stgPatchTop, err = stringExec("stg", "top"); err == nil {
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

func strInt(s string) int {
	r, _ := strconv.Atoi(s)
	return r
}
