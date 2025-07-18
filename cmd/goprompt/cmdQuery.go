package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
)

var (
	cmdQuery = &cobra.Command{
		Use:   "query",
		Short: "start the query that pulls data for the prompt",
	}
	flgQCmdStatus = cmdQuery.PersistentFlags().String(
		"cmd-status", "0",
		"cmd status of previous command (int)",
	)
	flgQPreexecTS = cmdQuery.PersistentFlags().String(
		"preexec-ts", "0",
		"pre-execution timestamp to gauge how log execution took (int)",
	)
	flgQTimeout = cmdQuery.PersistentFlags().Duration(
		"timeout", 0,
		"timeout after which to give up",
	)
	flgQPidParentSkip = cmdQuery.PersistentFlags().Int(
		"pid-parent-skip", 0,
		"skip this many parent PIDs when determining true parent process (when run from ZSH ZLE descriptor we end up with extra PID nesting)",
	)
	flgQPidChain = cmdQuery.PersistentFlags().Bool(
		"pid-chain", false,
		"add extra pid parent chain information",
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

	_partOS = "os_name"

	_partWorkDir      = "wd"
	_partWorkDirShort = "wd_trim"

	_partPid            = "pid"
	_partPidShell       = "pid_shell"
	_partPidShellExec   = "pid_shell_exec"
	_partPidShellApp    = "pid_shell_app"
	_partPidShellArgs   = "pid_shell_args"
	_partPidParent      = "pid_parent"
	_partPidParentExec  = "pid_parent_exec"
	_partPidParentApp   = "pid_parent_app"
	_partPidParentArgs  = "pid_parent_args"
	_partPidRemote      = "pid_remote"
	_partPidRemoteExec  = "pid_remote_exec"
	_partPidChain       = "pid_chain"
	_partPidChainLength = "pid_chain_length"

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

	_partVcsGitRebaseOp   = "vcs_git_rebase_op"
	_partVcsGitRebaseLeft = "vcs_git_rebase_op_left"

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

	prevCMDStatus := trim(*flgQCmdStatus)
	if prevCMDStatus != "0" {
		printPart(_partStatus, fmt.Sprintf("%s", prevCMDStatus))
	}

	osName := runtime.GOOS
	printPart(_partOS, osName)

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

		preexecTS := trim(*flgQPreexecTS)
		if preexecTS != "0" {
			if ts, err := strconv.Atoi(preexecTS); err == nil {
				cmdTS := time.Unix(int64(ts), 0)
				diff := nowTS.Sub(cmdTS).Round(time.Second)
				if diff > 1 {
					printPart(_partDuration, diff)
				}
			}
		}

		return nil
	})

	tasks.Go(func(ctx context.Context) error {
		type list []interface{}
		type dict map[string]interface{}

		psRef, err := process.NewProcess(int32(os.Getpid()))
		if err != nil {
			printPart("debug_ps_error", err.Error())
		}

		// Construct chain of processes
		psChain := make([]*process.Process, 0)
		for psRef != nil && psRef.Pid != 0 {
			psParent, err := psRef.ParentWithContext(ctx)
			if err != nil {
				printPart("debug_ps_error", err.Error())
				break
			}

			psChain = append(psChain, psParent)
			psRef = psParent
		}

		printPart(_partPidChainLength, len(psChain))

		var pidRemote *process.Process
		var pidChain list
		for psIdx, ps := range psChain {
			name, err := ps.Name()
			if err != nil {
				continue
			}
			cmdline, err := ps.CmdlineSlice()
			if err != nil {
				continue
			}

			// Find if we are in a remote session.
			if strings.Contains(name, "ssh") && pidRemote == nil {
				pidRemote = ps
			}

			psIdxAdj := psIdx - *flgQPidParentSkip

			pidExec := ""
			if len(cmdline) > 0 {
				pidExec = filepath.Base(cmdline[0])
			}
			pidApp := pidExec

			if osName == "darwin" {
				// Extract $SOME_LOCATION/$NAME.app/.../$EXEC_NAME from cmdline
				parts := strings.Split(cmdline[0], "/")
				for i := range parts[:len(parts)-1] {
					if strings.HasSuffix(parts[i], ".app") {
						pidApp = parts[i]
						pidExec = parts[len(parts)-1]
						break
					}
				}
			}

			pidChain = append(pidChain, dict{
				"name":    name,
				"pid":     ps.Pid,
				"cmdline": cmdline,
				"exec":    pidExec,
				"app":     pidApp,
			})

			if psIdxAdj == 1 {
				printPart(_partPidShell, ps.Pid)
				printPart(_partPidShellExec, pidExec)
				printPart(_partPidShellApp, pidApp)
				printPart(_partPidShellArgs, name)
			} else if psIdxAdj == 2 {
				printPart(_partPidParent, ps.Pid)
				printPart(_partPidParentExec, pidExec)
				printPart(_partPidParentApp, pidApp)
				printPart(_partPidParentArgs, name)
			}
		}

		if pidRemote != nil {
			name, err := pidRemote.Name()
			if err == nil {
				pidShellRemoteExecName, _, _ := strings.Cut(name, " ")
				printPart(_partPidRemote, pidRemote.Pid)
				printPart(_partPidRemoteExec, pidShellRemoteExecName)
			}
		}

		if *flgQPidChain {
			pidChainJson, _ := json.Marshal(pidChain)
			printPart(_partPidChain, string(pidChainJson))
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

		gitDir, _ := stringExec("git", "rev-parse", "--path-format=absolute", "--git-dir")

		subTasks.Go(func(ctx context.Context) error {
			headRef := ""
			if cherryHeadB, _ := os.ReadFile(filepath.Join(gitDir, "CHERRY_PICK_HEAD")); len(cherryHeadB) > 0 {
				headRef = trim(string(cherryHeadB))
				printPart(_partVcsGitRebaseOp, "cherry")
			} else if mergeHeadB, _ := os.ReadFile(filepath.Join(gitDir, "MERGE_HEAD")); len(mergeHeadB) > 0 {
				headRef = trim(string(mergeHeadB))
				printPart(_partVcsGitRebaseOp, "merge")
			} else if rebaseHeadB, _ := os.ReadFile(filepath.Join(gitDir, "rebase-merge", "orig-head")); len(rebaseHeadB) > 0 {
				headRef = trim(string(rebaseHeadB))
				printPart(_partVcsGitRebaseOp, "rebase")

				actionsLeftB, _ := os.ReadFile(filepath.Join(gitDir, "rebase-merge", "git-rebase-todo"))
				actionsLeft := trim(string(actionsLeftB))
				if len(actionsLeftB) == 0 {
					printPart(_partVcsGitRebaseLeft, 1)
				} else {
					printPart(_partVcsGitRebaseLeft, len(strings.Split(string(actionsLeft), "\n"))+1)
				}
			}

			branch := ""

			if len(headRef) != 0 {
				branch, _ = stringExec("git", "name-rev", "--name-only", headRef)
			} else {
				branch, _ = stringExec("git", "branch", "--show-current")
			}
			printPart(_partVcsBranch, branch)

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

func jsonPart(d interface{}) string {
	b, _ := json.Marshal(d)
	return string(b)
}
