package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
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
)

func init() {
	cmdQuery.RunE = cmdQueryRun
}

func cmdQueryRun(_ *cobra.Command, _ []string) error {
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

			diff := nowTS.Sub(cmdTS).Round(time.Second)
			if diff > 1 {
				printPart("ds", diff)
			}
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
					//printPart("vcs_dirty_st", js(status))
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
