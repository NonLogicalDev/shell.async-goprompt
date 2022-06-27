package shellout

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

type Cmd struct {
	path string
	args []string

	stdin  io.Reader
	stderr io.Writer
	stdout io.Writer

	env map[string]string

	cmd *exec.Cmd
}

type Option func(ex *Cmd)

func Passthrough() Option {
	return func(ex *Cmd) {
		ex.stdin = os.Stdin
		ex.stdout = os.Stdout
		ex.stderr = os.Stderr
	}
}

func Args(path string, args ...string) Option {
	return func(ex *Cmd) {
		ex.path = path
		ex.args = args
	}
}

func ArgsAdd(args ...string) Option {
	return func(ex *Cmd) {
		ex.args = append(ex.args, args...)
	}
}

func ArgsAddFN(fn func() (args []string)) Option {
	return func(ex *Cmd) {
		ex.args = append(ex.args, fn()...)
	}
}

func EnvSet(env map[string]string) Option {
	return func(ex *Cmd) {
		ex.env = env
	}
}

func BindStderr(b io.Writer) Option {
	return func(ex *Cmd) {
		ex.stderr = b
	}
}

func BindStdout(b io.Writer) Option {
	return func(ex *Cmd) {
		ex.stdout = b
	}
}

func BindStdin(b io.Reader) Option {
	return func(ex *Cmd) {
		ex.stdin = b
	}
}

func (ex *Cmd) Args() []string {
	return ex.args
}

func (ex *Cmd) Ref() *exec.Cmd {
	return ex.cmd
}

func (ex *Cmd) RunBytes() ([]byte, error) {
	ex.stdout = nil
	return ex.cmd.Output()
}

func (ex *Cmd) RunString() (string, error) {
	output, err := ex.RunBytes()
	return string(output), err
}

func (ex *Cmd) Run() error {
	return ex.cmd.Run()
}

func (ex *Cmd) Pipe(other *Cmd, tee bool) error {
	oldStdout := ex.cmd.Stdout

	ex.cmd.Stdout = nil
	pipe, err := ex.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if tee && oldStdout != nil {
		teePipe := io.TeeReader(pipe, oldStdout)
		other.cmd.Stdin = teePipe
	} else {
		other.cmd.Stdin = pipe
	}
	return nil
}

func New(ctx context.Context, options ...Option) *Cmd {
	ex := &Cmd{
		env: map[string]string{},
	}
	for _, opt := range options {
		opt(ex)
	}

	cmd := exec.CommandContext(ctx, ex.path, ex.args...)
	cmd.Stdin = ex.stdin
	cmd.Stdout = ex.stdout
	cmd.Stderr = ex.stderr

	for k, v := range ex.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	ex.cmd = cmd

	return ex
}

type CmdPipe struct {
	cmds []*exec.Cmd
}

func Pipe(tee bool, cmds ...*Cmd) error {
	for i := 1; i < len(cmds); i++ {
		prevCMD := cmds[i-1]
		currCMD := cmds[i]

		err := prevCMD.Pipe(currCMD, tee)
		if err != nil {
			return err
		}
	}
	return nil
}

func RunAll(cmds ...*Cmd) []error {
	wg := new(sync.WaitGroup)
	errCH := make(chan error)

	wg.Add(len(cmds))
	for _, cmd := range cmds {
		go func(cmd *Cmd) {
			defer wg.Done()
			errCH <- cmd.Run()
		}(cmd)
	}
	go func() {
		wg.Wait()
		close(errCH)
	}()

	var errs []error
	for err := range errCH {
		errs = append(errs, err)
	}
	return errs

}
