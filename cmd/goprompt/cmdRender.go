package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

var (
	cmdRender = &cobra.Command{
		Use:   "render",
		Short: "render the prompt based on the results of query",
	}

	flgREscapeMode = cmdRender.PersistentFlags().String(
		"escape-mode", "none",
		"color / escape rendering mode of the prompt (zsh, ascii, none)",
	)

	flgRLoading = cmdRender.PersistentFlags().Bool(
		"prompt-loading", false,
		"notify that prompt query is ongoing",
	)
	flgRMode = cmdRender.PersistentFlags().String(
		"prompt-mode", "normal",
		"mode of the prompt (normal, edit)",
	)
	flgRPromptStartMark = cmdRender.PersistentFlags().String(
		"prompt-mark-start", "",
		"mark to place at the start of the prompt (first prompt line)",
	)
)

func init() {
	cmdRender.RunE = cmdRenderRun
}

var (
	redC     = fmt.Sprint
	greenC   = fmt.Sprint
	yellowC  = fmt.Sprint
	greyC    = fmt.Sprint
	blueC    = fmt.Sprint
	magentaC = fmt.Sprint
	normalC  = fmt.Sprint
	newline  = "\n"
)

func setColorMode(mode string) {
	wrapC := func(pref, suff string) func(args ...interface{}) string {
		return func(args ...interface{}) string {
			return pref + fmt.Sprint(args...) + suff
		}
	}

	if mode == "zsh" {
		redC = wrapC("%F{red}", "%F{reset}")
		greenC = wrapC("%F{green}", "%F{reset}")
		yellowC = wrapC("%F{yellow}", "%F{reset}")
		blueC = wrapC("%F{blue}", "%F{reset}")
		magentaC = wrapC("%F{magenta}", "%F{reset}")
		greyC = wrapC("%F{black}", "%F{reset}")
		newline = "\n%{\r%}"

	} else if mode == "ascii" {
		redC = color.Red.Render
		greenC = color.Green.Render
		yellowC = color.Yellow.Render
		blueC = color.Blue.Render
		magentaC = color.Magenta.Render
		greyC = color.Black.Render
		newline = "\n"
	}
}

func cmdRenderRun(_ *cobra.Command, _ []string) error {
	setColorMode(*flgREscapeMode)

	if _, err := os.Stdin.Stat(); err != nil {
		fmt.Printf("%#v", err)
	}

	out, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	lines := strings.Split(string(out), "\n")
	p := make(map[string]string)
	for _, line := range lines {
		key, value, ok := strings.Cut(line, "\t")
		if ok {
			p[key] = value
		}
	}

	var partsTop []string
	if p[_partVcs] == "git" {
		var gitParts []string

		gitMark := "git"
		gitMarkC := yellowC

		gitBranch := fmt.Sprint(p[_partVcsBranch])
		gitBranchC := greenC

		gitDirtyMarks := ""
		gitDirtyMarksC := redC
		if p[_partVcsDirty] != "" && p[_partVcsDirty] != "0" {
			gitDirtyMarks = "&"

			if p[_partVcsGitIdxExcluded] == "0" {
				gitDirtyMarksC = greenC
			}
		}

		distanceMarks := ""
		distanceMarksC := magentaC

		distanceAhead := strInt(p[_partVcsLogAhead])
		distanceBehind := strInt(p[_partVcsLogBehind])
		if distanceAhead > 0 || distanceBehind > 0 {
			distanceMarks = fmt.Sprintf("[+%v:-%v]", distanceAhead, distanceBehind)
		}

		rebaseOp := ""
		rebaseOpC := redC
		if len(p[_partVcsGitRebaseOp]) != 0 {
			rebaseOp = p[_partVcsGitRebaseOp]
			if p[_partVcsGitRebaseLeft] != "" {
				rebaseOp += fmt.Sprintf("(%v)", p[_partVcsGitRebaseLeft])
			}
		}

		gitParts = append(gitParts, gitMarkC(gitMark))
		gitParts = append(gitParts, gitBranchC(gitBranch))
		if len(gitDirtyMarks) > 0 {
			gitParts = append(gitParts, gitDirtyMarksC(gitDirtyMarks))
		}
		if len(distanceMarks) > 0 {
			gitParts = append(gitParts, distanceMarksC(distanceMarks))
		}
		if len(rebaseOp) > 0 {
			gitParts = append(gitParts, rebaseOpC(rebaseOp))
		}

		partsTop = append(partsTop, fmt.Sprintf("{%v}", strings.Join(gitParts, ":")))
	}

	if p[_partVcs] == "sapling" {
		var saplParts []string

		saplMark := "spl"
		saplMarkC := yellowC

		saplBookmark := fmt.Sprint(p[_partVcsSaplBookmarkActive])
		saplBookmarkC := greenC

		saplDirtyMarks := ""
		saplDirtyMarksC := redC
		if p[_partVcsDirty] != "" && p[_partVcsDirty] != "0" {
			saplDirtyMarks = "&"
		}

		saplParts = append(saplParts, saplMarkC(saplMark))
		saplParts = append(saplParts, saplBookmarkC(saplBookmark))
		if len(saplDirtyMarks) > 0 {
			saplParts = append(saplParts, saplDirtyMarksC(saplDirtyMarks))
		}

		partsTop = append(partsTop, fmt.Sprintf("{%v}", strings.Join(saplParts, ":")))
	}

	if p[_partVcsStg] != "" {
		var stgParts []string

		stgMark := "stg"
		stgMarkC := yellowC

		stgTopPatch := p[_partVcsStgTop]
		stgTopPatchC := greenC

		stgQueueMark := ""
		stgQueueMarkC := normalC

		stgQueueLen := strInt(p[_partVcsStgQlen])
		stgQueuePos := strInt(p[_partVcsStgQpos])
		if stgQueuePos > 0 {
			stgQueueMark = fmt.Sprintf("%d/%d", stgQueuePos, stgQueueLen)
		}

		if strInt(p[_partVcsStgDirty]) != 0 {
			stgTopPatchC = redC
		}

		stgParts = append(stgParts, stgMarkC(stgMark))

		if len(stgTopPatch) > 0 {
			stgParts = append(stgParts, stgTopPatchC(stgTopPatch))

		}

		if len(stgQueueMark) > 0 {
			stgParts = append(stgParts, stgQueueMarkC(stgQueueMark))
		}

		partsTop = append(partsTop, fmt.Sprintf("{%v}", strings.Join(stgParts, ":")))
	}

	var partsBottom []string
	if strInt(p[_partStatus]) > 0 {
		partsBottom = append(partsBottom, redC("["+p[_partStatus]+"]"))
	}

	if p[_partPidParentExec] != "" {
		partsBottom = append(partsBottom, "("+p[_partPidParentExec]+")")
	}

	partsBottom = append(partsBottom, yellowC("(")+blueC(p[_partWorkDirShort])+yellowC(")"))

	if p[_partDuration] != "" {
		partsBottom = append(partsBottom, fmt.Sprintf("%v", p[_partDuration]))
	}

	nowTS := time.Now()
	cmdTS := timeFMT(nowTS)
	if len(p[_partTimestamp]) != 0 {
		cmdTS = p[_partTimestamp]
	}
	partsBottom = append(partsBottom, fmt.Sprintf("[%v]", cmdTS))

	if len(p[_partPidRemote]) != 0 {
		partsBottom = append(partsBottom, greyC(fmt.Sprintf("%v@%v", p[_partSessionUsername], p[_partSessionHostname])))
	}

	promptMarker := magentaC(">")
	if *flgRMode == "edit" {
		promptMarker = redC("<")
	}

	promptStatusMarker := ":? "
	if status, ok := p["done"]; ok {
		if status == "ok" {
			promptStatusMarker = ":: "
		} else if status == "timeout" {
			promptStatusMarker = "xx "
		}
	}

	promptLines := []string{""}
	if len(partsTop) > 0 {
		promptLines = append(promptLines, promptStatusMarker+strings.Join(partsTop, " "))
	} else {
		promptLines = append(promptLines, promptStatusMarker+strings.Repeat("-", 30))
	}
	if len(partsBottom) > 0 {
		promptLines = append(promptLines, promptStatusMarker+strings.Join(partsBottom, " "))
	}
	promptLines = append(promptLines, promptMarker)

	// Add prompt mark to last line
	lastLine := len(promptLines) - 1
	if lastLine >= 0 {
		promptLines[lastLine] = fmt.Sprintf("%v%v", *flgRPromptStartMark, promptLines[lastLine])
	}

	fullPrompt := strings.Join(promptLines, newline)
	fmt.Print(fullPrompt)

	return nil
}
