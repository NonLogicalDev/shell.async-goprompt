package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var bgctx, bgctxCancel = context.WithCancel(context.Background())

var (
	cmd = &cobra.Command{
		Use: "goprompt",
	}
	envLogFile = os.Getenv("GOPROMPT_LOG_FILE")
)

func debugLog(msg string, args ...[]interface{}) {
	if len(envLogFile) == 0 {
		return
	}
	f, err := os.OpenFile(envLogFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	if _, err := f.WriteString(msg + "\n"); err != nil {
		return
	}

}

func init() {
	cmd.AddCommand(cmdQuery)
	cmd.AddCommand(cmdRender)
}

func main() {

	err := cmd.ExecuteContext(bgctx)
	if err != nil {
		panic(err)
	}
}
