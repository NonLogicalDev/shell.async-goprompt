package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NonLogicalDev/shell.async-goprompt/pkg/shellout"
)

type AsyncTaskDispatcher struct {
	wg sync.WaitGroup
}

func (d *AsyncTaskDispatcher) Dispatch(fn func()) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		fn()
	}()
}

func (d *AsyncTaskDispatcher) Wait() {
	d.wg.Wait()
}

//: ----------------------------------------------------------------------------

type shellKV struct {
	name  string
	value interface{}
}

func (kv shellKV) Print() {
	fmt.Println(kv.String())
}

func (kv shellKV) String() string {
	return fmt.Sprintf("%s\t%v", kv.name, kv.value)
}

func shellKVStaggeredPrinter(
	printCH chan shellKV,

	dFirst time.Duration,
	d time.Duration,
) {
	var parts []shellKV

	printParts := func(parts []shellKV) {
		for _, p := range parts {
			p.Print()
		}
		if len(parts) > 0 {
			fmt.Println()
		}
		os.Stdout.Sync()
	}

	timerFirst := time.NewTimer(dFirst)
	timer := time.NewTimer(d)

LOOP:
	for {
		select {
		case rx, ok := <-printCH:
			if !ok {
				break LOOP
			}
			parts = append(parts, rx)

		case <-timerFirst.C:
			printParts(parts)
			parts = nil

		case <-timer.C:
			printParts(parts)
			parts = nil
			timer.Reset(d)
		}
	}

	printParts(parts)
	parts = nil
}

//: ----------------------------------------------------------------------------

func stringExec(path string, args ...string) (string, error) {
	out, err := shellout.New(bgctx,
		shellout.Args(path, args...),
		shellout.EnvSet(map[string]string{
			"GIT_OPTIONAL_LOCKS": "0",
		}),
	).RunString()
	return trim(out), err
}

func toJSON(v interface{}) string {
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
	return strings.Trim(s, "\n")
}

func strInt(s string) int {
	r, _ := strconv.Atoi(s)
	return r
}
