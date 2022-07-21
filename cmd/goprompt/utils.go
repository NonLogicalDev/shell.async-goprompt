package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

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

func printPart(name string, value interface{}) {
	if _, err := os.Stdout.Stat(); err != nil {
		os.Exit(1)
	}
	fmt.Printf("%s\t%v\n", name, value)
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
