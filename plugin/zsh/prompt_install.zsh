# PROMPT_ASYNC_ZLE: ------------------------------------------------------------
if (($+commands[${GOPROMPT}])) || [[ -e ${GOPROMPT} ]]; then
	autoload -Uz promptinit
	promptinit && eval "$(${GOPROMPT} install zsh.plugin)"
fi
# ------------------------------------------------------------------------------