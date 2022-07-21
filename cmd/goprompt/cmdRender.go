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

	flgRLoading = cmdRender.PersistentFlags().Bool(
		"prompt-loading", false,
		"is prompt query not yet done rendering",
	)
	flgRColorMode = cmdRender.PersistentFlags().String(
		"color-mode", "none",
		"color rendering mode of the prompt (zsh, ascii, none)",
	)
	flgRMode = cmdRender.PersistentFlags().String(
		"prompt-mode", "normal",
		"mode of the prompt (normal, edit)",
	)

	// DEPRECATED
	flgRNewline = cmdRender.PersistentFlags().String(
		"newline", "\n",
		"newline for the prompt",
	)
)

var (
	redC     = fmt.Sprint
	greenC   = fmt.Sprint
	yellowC  = fmt.Sprint
	blueC    = fmt.Sprint
	magentaC = fmt.Sprint
	normalC  = fmt.Sprint
)

func setColorMode(mode string) {
	wrapC := func(pref, suff string) func (args ...interface{}) string {
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
	} else if mode == "ascii" {
		redC = color.Red.Render
		greenC = color.Green.Render
		yellowC = color.Yellow.Render
		blueC = color.Blue.Render
		magentaC = color.Magenta.Render
	}
}


func init() {
	cmdRender.RunE = cmdRenderRun
}

func cmdRenderRun(_ *cobra.Command, _ []string) error {
	setColorMode(*flgRColorMode)

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
	if p["vcs"] == "git" {
		var gitParts []string

		gitMark := "git"
		gitMarkC := yellowC

		gitBranch := fmt.Sprint(p["vcs_br"])
		gitBranchC := greenC

		gitDirtyMarks := ""
		gitDirtyMarksC := redC
		if p["vcs_dirty"] != "" && p["vcs_dirty"] != "0" {
			gitDirtyMarks = "&"
		}

		distanceMarks := ""
		distanceMarksC := magentaC

		distanceAhead := strInt(p["vcs_log_ahead"])
		distanceBehind := strInt(p["vcs_log_ahead"])
		if distanceAhead > 0 || distanceBehind > 0 {
			distanceMarks = fmt.Sprintf("[+%v:-%v]", distanceAhead, distanceBehind)
		}

		gitParts = append(gitParts, gitMarkC(gitMark))
		gitParts = append(gitParts, gitBranchC(gitBranch))
		if len(gitDirtyMarks) > 0 {
			gitParts = append(gitParts, gitDirtyMarksC(gitDirtyMarks))
		}
		if len(distanceMarks) > 0 {
			gitParts = append(gitParts, distanceMarksC(distanceMarks))
		}

		partsTop = append(partsTop, fmt.Sprintf("{%v}", strings.Join(gitParts, ":")))
	}

	if p["stg"] != "" {
		var stgParts []string

		stgMark := "stg"
		stgMarkC := yellowC

		stgTopPatch := p["stg_top"]
		stgTopPatchC := greenC

		stgQueueMark := ""
		stgQueueMarkC := normalC

		stgQueueLen := strInt(p["stg_qlen"])
		stgQueuePos := strInt(p["stg_qpos"])
		if stgQueuePos > 0 {
			stgQueueMark = fmt.Sprintf("%d/%d", stgQueuePos, stgQueueLen)
		}

		if strInt(p["stg_dirty"]) != 0 {
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
	if strInt(p["st"]) > 0 {
		partsBottom = append(partsBottom, redC("["+p["st"]+"]"))
	}

	if p["pid_parent_exec"] != "" {
		partsBottom = append(partsBottom, "("+p["pid_parent_exec"]+")")
	}

	partsBottom = append(partsBottom, yellowC("(")+blueC(p["wd_trim"])+yellowC(")"))

	if p["ds"] != "" {
		partsBottom = append(partsBottom, fmt.Sprintf("%v", p["ds"]))
	}

	nowTS := time.Now()
	cmdTS := timeFMT(nowTS)
	if len(p["ts"]) != 0 {
		cmdTS = p["ts"]
	}
	partsBottom = append(partsBottom, fmt.Sprintf("[%v]", cmdTS))

	promptMarker := magentaC(">")
	if *flgRMode == "edit" {
		promptMarker = redC("<")
	}

	promptStatusMarker := ":: "
	if *flgRLoading {
		promptStatusMarker = ":? "
	}

	promptLines := []string{"",
		promptStatusMarker + strings.Join(partsTop, " "),
		promptStatusMarker + strings.Join(partsBottom, " "),
		promptMarker,
	}

	fmt.Print(strings.Join(promptLines, "\n"))

	return nil
}
