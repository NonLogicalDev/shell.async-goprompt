# PROMPT_ASYNC_ZLE: ------------------------------------------------------------
if (($+commands[goprompt])); then
	autoload -Uz promptinit
	promptinit && eval "$(goprompt install zsh.plugin)"
fi
# ------------------------------------------------------------------------------