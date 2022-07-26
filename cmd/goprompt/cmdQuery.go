package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	ps "github.com/mitchellh/go-ps"
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

const (
	_partStatus    = "st"
	_partTimestamp = "ts"
	_partDuration  = "ds"

	_partWorkDir      = "wd"
	_partWorkDirShort = "wd_trim"

	_partPidShell      = "pid_shell"
	_partPidShellExec  = "pid_shell_exec"
	_partPidParent     = "pid_parent"
	_partPidParentExec = "pid_parent_exec"

	_partVcs       = "vcs"
	_partVcsBranch = "vcs_br"
	_partVcsDirty  = "vcs_dirty"

	_partVcsLogAhead  = "vcs_log_ahead"
	_partVcsLogBehind = "vcs_log_behind"

	_partVcsStg      = "stg"
	_partVcsStgQlen  = "stg_qlen"
	_partVcsStgQpos  = "stg_qpos"
	_partVcsStgTop   = "stg_top"
	_partVcsStgDirty = "stg_dirty"

	_partVcsGitIdxTotal    = "git_idx_total"
	_partVcsGitIdxIncluded = "git_idx_incl"
	_partVcsGitIdxExcluded = "git_idx_excl"
)

func init() {
	cmdQuery.RunE = cmdQueryRun
}

func timeFMT(ts time.Time) string {
	return ts.Format("15:04:05 01/02/06")
}

func cmdQueryRun(_ *cobra.Command, _ []string) error {
	tasks := new(AsyncTaskDispatcher)

	printCH := make(chan shellKV)
	printerWG := new(sync.WaitGroup)
	printerWG.Add(1)
	go func() {
		defer printerWG.Done()
		shellKVStaggeredPrinter(printCH, 20*time.Millisecond, 600*time.Millisecond)
	}()
	printerStop := func() {
		close(printCH)
		printerWG.Wait()
	}
	printPart := func(name string, value interface{}) {
		printCH <- shellKV{name, value}
	}

	nowTS := time.Now()
	printPart(_partTimestamp, timeFMT(nowTS))

	if *flgQCmdStatus != 0 {
		printPart(_partStatus, fmt.Sprintf("%#v", *flgQCmdStatus))
	}

	defer func() {
		tasks.Wait()
		printerStop()
	}()

	tasks.Dispatch(func() {
		homeDir := os.Getenv("HOME")

		if wd, err := os.Getwd(); err == nil {
			wdh := strings.Replace(wd, homeDir, "~", 1)

			printPart(_partWorkDir, wdh)
			printPart(_partWorkDirShort, trimPath(wdh))
		}

		if *flgQPreexecTS != 0 {
			cmdTS := time.Unix(int64(*flgQPreexecTS), 0)

			diff := nowTS.Sub(cmdTS).Round(time.Second)
			if diff > 1 {
				printPart(_partDuration, diff)
			}
		}
	})

	tasks.Dispatch(func() {
		pidCurr := os.Getpid()
		var pidShell ps.Process

		for i := 0; i < 3; i++ {
			var err error
			pidShell, err = ps.FindProcess(pidCurr)
			if err != nil {
				return
			}
			pidCurr = pidShell.PPid()
		}

		if pidShell == nil {
			return
		}

		printPart(_partPidShell, pidShell.Pid())
		printPart(_partPidShellExec, pidShell.Executable())

		pidShellParent, err := ps.FindProcess(pidShell.PPid())
		if err != nil {
			return
		}

		printPart(_partPidParent, pidShellParent.Pid())
		printPart(_partPidParentExec, pidShellParent.Executable())
	})

	tasks.Dispatch(func() {
		subTasks := new(AsyncTaskDispatcher)
		defer subTasks.Wait()

		if _, err := stringExec("git", "rev-parse", "--show-toplevel"); err == nil {
			printPart(_partVcs, "git")
		} else {
			return
		}

		subTasks.Dispatch(func() {
			if branch, err := stringExec("git", "branch", "--show-current"); err == nil {
				branch = trim(branch)
				if len(branch) > 0 {
					printPart(_partVcsBranch, trim(branch))
					return
				}
			}

			if branch, err := stringExec("git", "name-rev", "--name-only", "HEAD"); err == nil {
				branch = trim(branch)
				if len(branch) > 0 {
					printPart(_partVcsBranch, trim(branch))
					return
				}
			}
		})

		subTasks.Dispatch(func() {
			status, err := stringExec("git", "status", "--porcelain")
			if err != nil {
				return
			}

			if len(status) == 0 {
				printPart(_partVcsDirty, 0)
				return
			}

			printPart(_partVcsDirty, 1)

			fTotal := 0
			fInIndex := 0
			fOutOfIndex := 0

			lines := strings.Split(status, "\n")
			for _, line := range lines {
				if len(line) < 2 {
					continue
				}

				statusInIndex := line[0]
				statusOutOfIndex := line[1]

				if statusInIndex != ' ' {
					fInIndex += 1
				}
				if statusOutOfIndex != ' ' {
					fOutOfIndex += 1
				}

				fTotal += 1
			}

			printPart(_partVcsGitIdxTotal, fTotal)
			printPart(_partVcsGitIdxIncluded, fInIndex)
			printPart(_partVcsGitIdxExcluded, fOutOfIndex)
		})

		subTasks.Dispatch(func() {
			if status, err := stringExec("git", "rev-list", "--left-right", "--count", "HEAD...@{u}"); err == nil {
				parts := strings.SplitN(status, "\t", 2)
				if len(parts) < 2 {
					parts = []string{"0", "0"}
				}

				printPart(_partVcsLogAhead, parts[0])
				printPart(_partVcsLogBehind, parts[1])
			}
		})
	})

	tasks.Dispatch(func() {
		var err error

		subTasks := new(AsyncTaskDispatcher)
		defer subTasks.Wait()

		var stgSeriesLen string
		if stgSeriesLen, err = stringExec("stg", "series", "-c"); err == nil {
			printPart(_partVcsStg, "1")
			printPart(_partVcsStgQlen, stgSeriesLen)
		} else {
			return
		}

		subTasks.Dispatch(func() {
			if stgSeriesPos, err := stringExec("stg", "series", "-cA"); err == nil {
				printPart(_partVcsStgQpos, stgSeriesPos)
			}
		})

		var stgPatchTop string
		if stgPatchTop, err = stringExec("stg", "top"); err == nil {
			printPart(_partVcsStgTop, stgPatchTop)
		} else {
			return
		}

		subTasks.Dispatch(func() {
			gitSHA, _ := stringExec("stg", "id")
			stgSHA, _ := stringExec("stg", "id", stgPatchTop)

			if gitSHA != stgSHA {
				printPart(_partVcsStgDirty, 1)
			} else {
				printPart(_partVcsStgDirty, 0)
			}
		})
	})

	return nil
}
