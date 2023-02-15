# In a file `prompt_asynczle_setup` available on `fpath`
emulate -L zsh

typeset -g C_PROMPT_NEWLINE=$'\n%{\r%}'

typeset -g G_LAST_STATUS=0
typeset -g G_PREEXEC_TS=0
typeset -g G_ASYNC_DONE=0

typeset -g G_PROMPT_DATA=""

typeset -g G_LAST_PROMPT=""

declare -gA ZLE_ASYNC_FDS=()
declare -gA ZLE_ASYNC_PIDS=()

ZSH_ASYNC_TIMEOUT=${ZSH_ASYNC_TIMEOUT:-0}

#-------------------------------------------------------------------------------

__async_prompt_query() {
  if ! (( $+commands[goprompt] )); then
    echo -n ""
    return
  fi

  goprompt query \
    --cmd-status "$G_LAST_STATUS" \
    --preexec-ts "$G_PREEXEC_TS"
}

__async_prompt_render() {
  if ! (( $+commands[goprompt] )); then
    echo -n "?>"
    return
  fi

  local MODE="normal"
  if [[ $KEYMAP == "viins" ]]; then
    MODE="edit"
  fi

  goprompt render \
    --prompt-mode "$MODE" \
    --color-mode "zsh"
}

#-------------------------------------------------------------------------------

__prompt_rerender() {
  local BR=$C_PROMPT_NEWLINE

  PROMPT="$(printf "%s\n" "$G_PROMPT_DATA" | __async_prompt_render) "

  if [[ $PROMPT != $G_LAST_PROMPT ]]; then
    zle && zle reset-prompt
  fi

  G_LAST_PROMPT="$PROMPT"
}

#-------------------------------------------------------------------------------
# Command Handlers + Async Comm
#-------------------------------------------------------------------------------

__prompt_preexec() {
    typeset -g G_PREEXEC_TS=$EPOCHSECONDS
}

__prompt_precmd() {
  # save the status of last command.
  G_LAST_STATUS=$?

  # reset prompt state
  G_PROMPT_DATA=""

  # set prompt status to rendering
  G_ASYNC_DONE=0

  __zle_async_dispatch __zle_async_fd_handler __async_prompt_query

  __prompt_rerender
}

#-------------------------------------------------------------------------------
# ZLE Async
#-------------------------------------------------------------------------------

__zle_async_fd_handler() {
  # NOTES: For my sanity, and for the curious:
  # Nothing in this function should block, if you want to have smooth prompt rendering experience.
  local ZLE_FD=$1

  # read in all data that is available
  if ! IFS=$'\n' read -r ASYNC_RESULT <&"$ZLE_FD"; then
    # select marks this fd if we reach EOF,
    # so handle this specially.
    __zle_async_detach "$ZLE_FD"
    G_ASYNC_DONE=1

    G_PROMPT_DATA="${G_PROMPT_DATA}"$'\n'"${ASYNC_RESULT}"
    __prompt_rerender

    return 1
  fi

  G_PROMPT_DATA="${G_PROMPT_DATA}"$'\n'"${ASYNC_RESULT}"
  if [[ $ASYNC_RESULT == "" ]]; then
    __prompt_rerender
  fi
}

__zle_async_dispatch() {
  local dispatch_handler="$1"; shift 1
  local command=( "$@" )

  # Close existing file descriptor for this handler.
  local OLD_ZLE_FD=${ZLE_ASYNC_FDS["${dispatch_handler}"]}
  local OLD_ZLE_PID=${ZLE_ASYNC_PIDS["${dispatch_handler}"]}
  if [[ -n $OLD_ZLE_FD ]]; then
    __zle_async_detach "$OLD_ZLE_FD" 2>/dev/null
  fi
  if [[ -n $OLD_ZLE_PID ]]; then
    # echo "$$: KILLING(2): $OLD_ZLE_PID" > ~/.goprompt
    kill -2 $OLD_ZLE_PID  2>/dev/null || true
  fi

  # Create File Descriptor and attach to async command
  # exec {ZLE_FD}< <( "${command[@]}" )
  __launch_coproc ZLE_FD ZLE_PID "${command[@]}"

  # Attach file a ZLE handler to file descriptor.
  zle -F $ZLE_FD "${dispatch_handler}"
  ZLE_ASYNC_FDS["${dispatch_handler}"]="$ZLE_FD"
  ZLE_ASYNC_PIDS["${dispatch_handler}"]="$ZLE_PID"
}

__zle_async_detach() {
  local ZLE_FD=$1
  # Close stdout.
  exec {ZLE_FD}<&-
  # Close the file-descriptor.
  zle -F "$ZLE_FD"
}

__launch_coproc() {
	local FD_VAR=$1
	local PID_VAR=$2
	shift 2

	local command=( "$@" )

	local tmpdir=$(mktemp -d)
	mkfifo "$tmpdir/pipe.proc"

	local output_fd
	exec {output_fd}< <(
		{ sh -c 'echo $PPID' && : } >"$tmpdir/pipe.proc";
		"${command[@]}"
	)
	local coproc_pid=$(cat <"$tmpdir/pipe.proc")

	command rm -rf "$tmpdir"

	eval "$FD_VAR='$output_fd'"
	eval "$PID_VAR='$coproc_pid'"
}

#-------------------------------------------------------------------------------

prompt_asynczle_setup() {
  autoload -Uz +X add-zsh-hook 2>/dev/null
  autoload -Uz +X add-zle-hook-widget 2>/dev/null

  add-zsh-hook precmd  __prompt_precmd
  add-zsh-hook preexec __prompt_preexec

  zle -N __prompt_rerender
  if (( $+functions[add-zle-hook-widget] )); then
    add-zle-hook-widget zle-line-finish __prompt_rerender
    add-zle-hook-widget zle-keymap-select __prompt_rerender
  fi
}

prompt_asynczle_setup "$@"
