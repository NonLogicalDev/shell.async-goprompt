package shellout

import (
	"bytes"
	"context"
	"testing"
)

func TestStuff(t *testing.T) {
	ctx := context.Background()

	cmdLsOut := bytes.Buffer{}
	cmdLs := New(ctx,
		Args("bash", "-xe", "-c", "sleep 1; pwd; echo \"Hello\""),
		Passthrough(), BindStdout(&cmdLsOut))

	cmdSedOut := bytes.Buffer{}
	cmdSed := New(ctx,
		Args("sed", "s:/:|:g"),
		Passthrough(), BindStdout(&cmdSedOut))

	cmdPerlOut := bytes.Buffer{}
	cmdPerl := New(ctx,
		Args("perl", "-pe", "s/(\\w+)/[\\1]/g"),
		Passthrough(), BindStdout(&cmdPerlOut))

	Pipe(true, cmdLs, cmdSed, cmdPerl)
	errs := RunAll(cmdLs, cmdSed, cmdPerl)

	t.Log(">>> errs")
	t.Log(errs)
	t.Log(">>> pwd")
	t.Log(cmdLsOut.String())
	t.Log(">>> sed")
	t.Log(cmdSedOut.String())
	t.Log(">>> perl")
	t.Log(cmdPerlOut.String())
}
