package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/NonLogicalDev/shell.async-goprompt/pkg/shellout"
	ps "github.com/mitchellh/go-ps"
)

// ----------------------------------------------------------------------------

func timeFMT(ts time.Time) string {
	return ts.Format("15:04:05 01/02/06")
}

// ----------------------------------------------------------------------------

type shellKV struct {
	name  string
	value interface{}
}

func (kv shellKV) String() string {
	return fmt.Sprintf("%s\t%v", kv.name, kv.value)
}

// ----------------------------------------------------------------------------

func shellKVStaggeredPrinter(
	printCH <-chan shellKV,

	dFirst time.Duration,
	d time.Duration,
) {
	var parts []shellKV

	printParts := func(parts []shellKV) {
		if len(parts) == 0 {
			return
		}
		for _, p := range parts {
			fmt.Println(p.String())
		}
		if len(parts) > 0 {
			fmt.Println()
		}
		os.Stdout.Sync()
	}

	timer := time.NewTimer(dFirst)

printLoop:
	for {
		select {
		case rx, ok := <-printCH:
			if !ok {
				break printLoop
			}
			parts = append(parts, rx)

		case <-timer.C:
			printParts(parts)
			parts = nil
			timer.Reset(d)
		}
	}

	printParts(parts)
	parts = nil
}

// ----------------------------------------------------------------------------

func stringExec(path string, args ...string) (string, error) {
	ctx, ctxCancel := context.WithTimeout(bgctx, 10*time.Second)
	defer ctxCancel()

	out, err := shellout.New(ctx,
		shellout.Args(path, args...),
		shellout.EnvSet(map[string]string{
			"GIT_OPTIONAL_LOCKS": "0",
		}),
	).RunString()

	return trim(out), err
}

func moduleFindProcessChain() ([]ps.Process, error) {
	psPTR := os.Getpid()
	var pidChain []ps.Process

	for {
		if psPTR == 0 {
			break
		}
		psInfo, err := ps.FindProcess(psPTR)
		if err != nil {
			return nil, err
		}
		pidChain = append(pidChain, psInfo)
		psPTR = psInfo.PPid()
	}

	return pidChain, nil
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
	return strings.Trim(s, "\n")
}

func strInt(s string) int {
	r, _ := strconv.Atoi(s)
	return r
}
