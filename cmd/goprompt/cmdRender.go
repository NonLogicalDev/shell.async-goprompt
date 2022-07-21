package main

import (
	"fmt"
	"github.com/gookit/color"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cmdRender = &cobra.Command{
		Use:   "render",
		Short: "render the prompt based on the results of query",
	}

	flgRIncomplete = cmdRender.PersistentFlags().Bool(
		"prompt-incomplete", false,
		"is prompt query done rendering",
	)
	flgRMode = cmdRender.PersistentFlags().String(
		"prompt-mode", "normal",
		"mode of the prompt (normal, edit)",
	)
	flgRNewline = cmdRender.PersistentFlags().String(
		"newline", "\n",
		"newline for the prompt",
	)
)

func init() {
	cmdRender.RunE = cmdRenderRun
}

func cmdRenderRun(_ *cobra.Command, _ []string) error {
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

		gitMark := fmt.Sprint("git")
		gitMarkC := color.Green.Render

		gitBranch := fmt.Sprint(p["vcs_br"])
		gitBranchC := color.Green.Render

		gitDirtyMarks := ""
		if p["vcs_dirty"] != "" && p["vcs_dirty"] != "0" {
			gitDirtyMarks = fmt.Sprint("&")
			gitMarkC = color.Yellow.Render
		}

		distanceMarks := ""
		distanceAhead := strInt(p["vcs_log_ahead"])
		distanceBehind := strInt(p["vcs_log_ahead"])
		if distanceAhead > 0 || distanceBehind > 0 {
			distanceMarks = fmt.Sprintf("[+%v:-%v]", distanceAhead, distanceBehind)
		}

		gitParts = append(gitParts, gitMarkC(gitMark))
		gitParts = append(gitParts, gitBranchC(gitBranch))
		if len(gitDirtyMarks) > 0 {
			gitParts = append(gitParts, gitDirtyMarks)
		}
		if len(distanceMarks) > 0 {
			gitParts = append(gitParts, distanceMarks)
		}

		partsTop = append(partsTop, fmt.Sprintf("{%v}", strings.Join(gitParts, ":")))
	}

	var partsBottom []string
	if strInt(p["st"]) > 0 {
		partsBottom = append(partsBottom, fmt.Sprintf("[%v]", p["st"]))
	}
	partsBottom = append(partsBottom, fmt.Sprintf("(%v)", p["wd"]))
	if p["ds"] != "" {
		partsBottom = append(partsBottom, fmt.Sprintf("%v", p["ds"]))
	}
	partsBottom = append(partsBottom, fmt.Sprintf("[%v]", p["ts"]))

	promptMarker := fmt.Sprint(">")
	if *flgRMode == "edit" {
		promptMarker = fmt.Sprint("<")
	}

	promptStatusMarker := ":: "
	if *flgRIncomplete {
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
