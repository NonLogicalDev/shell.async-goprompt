package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"EXP/pkg/shellout"
)

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

//: ----------------------------------------------------------------------------

type shellKV struct {
	name  string
	value interface{}
}

func (kv shellKV) Print() {
	fmt.Printf("%s\t%v\n", kv.name, kv.value)
}

func shellKVStaggeredPrinter(
	printCH chan shellKV,

	dFirst time.Duration,
	d time.Duration,
) {
	var parts []shellKV

	printParts := func() {
		for _, p := range parts {
			p.Print()
		}
		if len(parts) > 0 {
			fmt.Println()
		}

		parts = nil
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
			printParts()

		case <-timer.C:
			printParts()
			timer.Reset(d)
		}
	}

	printParts()
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
	return strings.Trim(s, "\n\t ")
}

func strInt(s string) int {
	r, _ := strconv.Atoi(s)
	return r
}
