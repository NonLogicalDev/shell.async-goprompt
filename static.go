package goprompt

import "embed"

var (
	//go:embed plugin/zsh
	ZSHPluginFiles embed.FS

	//go:embed plugin/fish
	FishPluginFiles embed.FS
)