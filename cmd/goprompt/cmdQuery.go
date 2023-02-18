package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"
	"time"

	ps "github.com/mitchellh/go-ps"
	"github.com/sourcegraph/conc/pool"
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
	flgQTimeout = cmdQuery.PersistentFlags().Duration(
		"timeout", 0,
		"timeout after which to give up",
	)
)

func init() {
	cmdQuery.RunE = cmdQueryRun
}

func mkWgPool() pool.ContextPool {
	return *pool.New().WithContext(bgctx)
}

const (
	_partStatus    = "st"
	_partTimestamp = "ts"
	_partDuration  = "ds"

	_partWorkDir      = "wd"
	_partWorkDirShort = "wd_trim"

	_partPid           = "pid"
	_partPidShell      = "pid_shell"
	_partPidShellExec  = "pid_shell_exec"
	_partPidParent     = "pid_parent"
	_partPidParentExec = "pid_parent_exec"
	_partPidRemote     = "pid_remote"
	_partPidRemoteExec = "pid_remote_exec"

	_partSessionUsername = "session_username"
	_partSessionHostname = "session_hostname"

	_partVcs       = "vcs"
	_partVcsBranch = "vcs_br"
	_partVcsDirty  = "vcs_dirty"

	_partVcsLogAhead  = "vcs_log_ahead"
	_partVcsLogBehind = "vcs_log_behind"

	_partVcsStg      = "vcs_git_stg"
	_partVcsStgQlen  = "vcs_git_stg_qlen"
	_partVcsStgQpos  = "vcs_git_stg_qpos"
	_partVcsStgTop   = "vcs_git_stg_top"
	_partVcsStgDirty = "vcs_git_stg_dirty"

	_partVcsGitIdxTotal    = "vcs_git_idx_total"
	_partVcsGitIdxIncluded = "vcs_git_idx_incl"
	_partVcsGitIdxExcluded = "vcs_git_idx_excl"

	_partVcsSaplRev             = "vcs_sapling_rev"
	_partVcsSaplNode            = "vcs_sapling_node"
	_partVcsSaplBookmarks       = "vcs_sapling_bookmarks"
	_partVcsSaplBookmarkActive  = "vcs_sapling_bookmarks_active"
	_partVcsSaplBookmarksRemote = "vcs_sapling_bookmarks_remote"
)

func handleQUIT() context.CancelFunc {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill, syscall.SIGTERM)

	defer debugLog("quit: terminating")

	// Stdout watchdog
	go func() {
		// debugLog("start watchdog " + fmt.Sprintf("%d", os.Getppid()))
		defer bgctxCancel()

		for {
			if _, err := os.Stdout.Stat(); err != nil {
				debugLog("quit: terminating early")
				return
			}

			tick := time.After(100 * time.Millisecond)
			select {
			case <-tick:
				continue
			case <-sig:
				debugLog("quit: terminating early")
				return
			case <-bgctx.Done():
				return
			}
		}
	}()

	return bgctxCancel
}

func cmdQueryRun(_ *cobra.Command, _ []string) error {
	debugLog("query: start")
	defer bgctxCancel()

	printerStop, printPart := startPrinter()
	defer printerStop()

	if *flgQTimeout != 0 {
		go func() {
			// Timeout handler
			select {
			case <-bgctx.Done():
				return
			case <-time.After(*flgQTimeout):
				printPart("done", "timeout")
				printerStop()
				bgctxCancel()
				os.Exit(1)
			}
		}()
	}

	tasks := mkWgPool()
	defer func() {
		tasks.Wait()
		printPart("done", "ok")
	}()

	nowTS := time.Now()
	printPart(_partTimestamp, timeFMT(nowTS))

	if *flgQCmdStatus != 0 {
		printPart(_partStatus, fmt.Sprintf("%#v", *flgQCmdStatus))
	}

	tasks.Go(func(ctx context.Context) error {
		homeDir := os.Getenv("HOME")

		if wd, err := os.Getwd(); err == nil {
			wdh := strings.Replace(wd, homeDir, "~", 1)

			printPart(_partWorkDir, wdh)
			printPart(_partWorkDirShort, trimPath(wdh))
		}

		sessionUser, err := user.Current()
		if err == nil {
			printPart(_partSessionUsername, sessionUser.Username)
		}

		sessionHostname, err := os.Hostname()
		if err == nil {
			printPart(_partSessionHostname, sessionHostname)
		}

		if *flgQPreexecTS != 0 {
			cmdTS := time.Unix(int64(*flgQPreexecTS), 0)

			diff := nowTS.Sub(cmdTS).Round(time.Second)
			if diff > 1 {
				printPart(_partDuration, diff)
			}
		}

		return nil
	})

	tasks.Go(func(_ context.Context) error {
		psChain, err := moduleFindProcessChain()
		if err != nil {
			return nil
		}

		if len(psChain) > 3 {
			pidShell := psChain[1]
			pidShellParent := psChain[2]

			pidShellExecName, _, _ := strings.Cut(pidShell.Executable(), " ")
			printPart(_partPidShell, pidShell.Pid())
			printPart(_partPidShellExec, pidShellExecName)

			pidShellParentExecName, _, _ := strings.Cut(pidShellParent.Executable(), " ")
			printPart(_partPidParent, pidShellParent.Pid())
			printPart(_partPidParentExec, pidShellParentExecName)
		}

		var pidRemote ps.Process
		for _, psInfo := range psChain {
			if strings.Contains(psInfo.Executable(), "ssh") {
				pidRemote = psInfo
				break
			}
		}
		if pidRemote != nil {
			pidShellRemoteExecName, _, _ := strings.Cut(pidRemote.Executable(), " ")
			printPart(_partPidRemote, pidRemote.Pid())
			printPart(_partPidRemoteExec, pidShellRemoteExecName)
		}

		return nil
	})

	tasks.Go(func(context.Context) error {
		subTasks := mkWgPool()
		defer subTasks.Wait()

		saplingTemplate := `{rev}\t{node}\t{join(remotenames, "#")}\t{join(bookmarks, "#")}\t{activebookmark}\t{ifcontains(rev, revset("."), "@")}\n`

		if _, err := stringExec("sl", "root"); err == nil {
			printPart(_partVcs, "sapling")
		} else {
			return nil
		}

		subTasks.Go(func(ctx context.Context) error {
			if revInfo, err := stringExec("sl", "log", "-r", ".", "--template", saplingTemplate); err == nil {
				info := strings.Split(revInfo, "\t")
				printPart(_partVcsSaplRev, info[0])
				printPart(_partVcsSaplNode, info[1])
				printPart(_partVcsSaplBookmarks, info[3])
				if info[4] == "" {
					printPart(_partVcsSaplBookmarkActive, "@")
				} else {
					printPart(_partVcsSaplBookmarkActive, info[4])
				}
				printPart(_partVcsSaplBookmarksRemote, info[2])
			}

			return nil
		})


		subTasks.Go(func(ctx context.Context) error {
			if saplStatus, err := stringExec("sl", "status"); err == nil {
				if len(saplStatus) == 0 {
					printPart(_partVcsDirty, 0)
					return nil
				}

				printPart(_partVcsDirty, 1)
			}
			return nil
		})

		return nil
	})

	tasks.Go(func(context.Context) error {
		subTasks := mkWgPool()
		defer subTasks.Wait()

		if _, err := stringExec("git", "rev-parse", "--show-toplevel"); err == nil {
			printPart(_partVcs, "git")
		} else {
			return nil
		}

		subTasks.Go(func(ctx context.Context) error {
			if branch, err := stringExec("git", "branch", "--show-current"); err == nil {
				branch = trim(branch)
				if len(branch) > 0 {
					printPart(_partVcsBranch, trim(branch))
					return nil
				}
			}

			if branch, err := stringExec("git", "name-rev", "--name-only", "HEAD"); err == nil {
				branch = trim(branch)
				if len(branch) > 0 {
					printPart(_partVcsBranch, trim(branch))
					return nil
				}
			}

			return nil
		})

		subTasks.Go(func(context.Context) error {
			status, err := stringExec("git", "status", "--porcelain")
			if err != nil {
				return nil
			}

			if len(status) == 0 {
				printPart(_partVcsDirty, 0)
				return nil
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

			return nil
		})

		subTasks.Go(func(context.Context) error {
			if status, err := stringExec("git", "rev-list", "--left-right", "--count", "HEAD...@{u}"); err == nil {
				parts := strings.SplitN(status, "\t", 2)
				if len(parts) < 2 {
					parts = []string{"0", "0"}
				}

				printPart(_partVcsLogAhead, parts[0])
				printPart(_partVcsLogBehind, parts[1])
			}
			return nil
		})

		return nil
	})

	tasks.Go(func(context.Context) error {
		var err error

		subTasks := mkWgPool()

		var stgSeriesLen string
		if stgSeriesLen, err = stringExec("stg", "series", "-c"); err == nil {
			printPart(_partVcsStg, "1")
			printPart(_partVcsStgQlen, stgSeriesLen)
		} else {
			return nil
		}

		subTasks.Go(func(context.Context) error {
			if stgSeriesPos, err := stringExec("stg", "series", "-cA"); err == nil {
				printPart(_partVcsStgQpos, stgSeriesPos)
			}
			return nil
		})

		var stgPatchTop string
		if stgPatchTop, err = stringExec("stg", "top"); err == nil {
			printPart(_partVcsStgTop, stgPatchTop)
		} else {
			return nil
		}

		subTasks.Go(func(context.Context) error {
			gitSHA, _ := stringExec("stg", "id")
			stgSHA, _ := stringExec("stg", "id", stgPatchTop)

			if gitSHA != stgSHA {
				printPart(_partVcsStgDirty, 1)
			} else {
				printPart(_partVcsStgDirty, 0)
			}
			return nil
		})

		return nil
	})

	return nil
}

func startPrinter() (func(), func(name string, value interface{})) {
	debugLog("query-printer: start")
	defer debugLog("query-printer: stop")

	printCH := make(chan shellKV)
	doneSIG := make(chan struct{})
	go func() {
		defer close(doneSIG)
		shellKVStaggeredPrinter(printCH, 20*time.Millisecond, 100*time.Millisecond)
	}()

	printerStop := func() {
		close(printCH)
		<-doneSIG
	}
	printPart := func(name string, value interface{}) {
		printCH <- shellKV{name, value}
	}

	printPart(_partPid, os.Getpid())
	return printerStop, printPart
}
