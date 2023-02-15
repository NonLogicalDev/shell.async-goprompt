package main

import (
	"context"
	"github.com/spf13/cobra"
	"os"
)

var bgctx, bgctxCancel = context.WithCancel(context.Background())

var (
	cmd = &cobra.Command{
		Use: "goprompt",
	}
)

func flog(msg string) {
	f, err := os.OpenFile(os.Getenv("HOME")+"/.goprompt",
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
