# In a file `prompt_asynczle_setup` available on `fpath`
emulate -L zsh

typeset -g ZSH_ASYNC_PROMPT_TIMEOUT=${ZSH_ASYNC_PROMPT_TIMEOUT:-5s}
typeset -g ZSH_ASYNC_PROMPT_EXEC=${GOPROMPT}

typeset -g ZSH_ASYNC_PROMPT_DATA=""
typeset -g ZSH_ASYNC_PROMPT_LAST=""

typeset -g ZSH_ASYNC_PROMPT_LAST_STATUS=0
typeset -g ZSH_ASYNC_PROMPT_PREEXEC_TS=0
typeset -g ZSH_ASYNC_PROMPT_QUERY_DONE=0

declare -gA __ZLE_ASYNC_FDS=()
typeset -g __ZSH_ASYNC_PROMPT_NEWLINE=$'\n%{\r%}'

#-------------------------------------------------------------------------------

__async_check_exec() {
  local exec_name="$1"
  if (( $+commands[${exec_name}] )); then
    return 0
  fi
  if [[ -e ${exec_name} ]]; then
    return 0
  fi
  return 1
}

__async_prompt_query() {
  if ! __async_check_exec "${ZSH_ASYNC_PROMPT_EXEC}"; then
    echo -n ""
    return
  fi

  ${ZSH_ASYNC_PROMPT_EXEC} query \
    --cmd-status "${ZSH_ASYNC_PROMPT_LAST_STATUS:-0}" \
    --preexec-ts "${ZSH_ASYNC_PROMPT_PREEXEC_TS:-0}" \
    --pid-parent-skip 1 \
    --timeout "${ZSH_ASYNC_PROMPT_TIMEOUT:-5s}"
}

__async_prompt_render() {
  if ! __async_check_exec "${ZSH_ASYNC_PROMPT_EXEC}"; then
    echo -n "?>"
    return
  fi

  local MODE="normal"
  if [[ $KEYMAP == "viins" ]]; then
    MODE="edit"
  fi

  local LOADING=1
  if [[ $ZSH_ASYNC_PROMPT_QUERY_DONE -eq 1 ]]; then
    LOADING=0
  fi

  ${ZSH_ASYNC_PROMPT_EXEC} render \
    --prompt-mode "$MODE" \
    --prompt-loading="$LOADING" \
    --color-mode "zsh"
}

#-------------------------------------------------------------------------------

__prompt_rerender() {
  PROMPT="$(printf "%s\n" "$ZSH_ASYNC_PROMPT_DATA" | __async_prompt_render) "

  if [[ $PROMPT != $ZSH_ASYNC_PROMPT_LAST ]]; then
    zle && zle reset-prompt
  fi

  ZSH_ASYNC_PROMPT_LAST="$PROMPT"
}

#-------------------------------------------------------------------------------
# Command Handlers + Async Comm
#-------------------------------------------------------------------------------

__prompt_preexec() {
    typeset -g ZSH_ASYNC_PROMPT_PREEXEC_TS=$EPOCHSECONDS
}

__prompt_precmd() {
  # save the status of last command.
  ZSH_ASYNC_PROMPT_LAST_STATUS=$?

  # reset prompt state
  ZSH_ASYNC_PROMPT_DATA=""

  # set prompt status to rendering
  ZSH_ASYNC_PROMPT_QUERY_DONE=0

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
    ZSH_ASYNC_PROMPT_QUERY_DONE=1

    ZSH_ASYNC_PROMPT_DATA="${ZSH_ASYNC_PROMPT_DATA}"$'\n'"${ASYNC_RESULT}"
    __prompt_rerender

    return 1
  fi

  ZSH_ASYNC_PROMPT_DATA="${ZSH_ASYNC_PROMPT_DATA}"$'\n'"${ASYNC_RESULT}"
  if [[ $ASYNC_RESULT == "" ]]; then
    __prompt_rerender
  fi
}

__zle_async_dispatch() {
  local dispatch_handler="$1"; shift 1
  local command=( "$@" )

  # Close existing file descriptor for this handler.
  local OLD_ZLE_FD=${__ZLE_ASYNC_FDS["${dispatch_handler}"]}
  if [[ -n $OLD_ZLE_FD ]]; then
    __zle_async_detach "$OLD_ZLE_FD" 2>/dev/null
  fi

  local ZLE_FD

  # Create File Descriptor and attach to async command
  exec {ZLE_FD}< <( "${command[@]}" )

  # Attach file a ZLE handler to file descriptor.
  zle -F $ZLE_FD "${dispatch_handler}"
  __ZLE_ASYNC_FDS["${dispatch_handler}"]="$ZLE_FD"
}

__zle_async_detach() {
  local ZLE_FD=$1
  # Close stdout.
  exec {ZLE_FD}<&-
  # Close the file-descriptor.
  zle -F "$ZLE_FD"
}


#-------------------------------------------------------------------------------

prompt_asynczle_setup() {
  zmodload zsh/datetime || :

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
