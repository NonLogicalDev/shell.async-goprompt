package main

import (
	"context"
	"github.com/spf13/cobra"
)

var bgctx = context.Background()

var (
	cmd = &cobra.Command{
		Use: "goprompt",
	}
)

func init() {
	cmd.AddCommand(cmdQuery)
	cmd.AddCommand(cmdRender)
}

func main() {
	err := cmd.ExecuteContext(context.Background())
	if err != nil {
		panic(err)
	}
}
