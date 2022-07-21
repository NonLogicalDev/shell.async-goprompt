# In a file `prompt_asynczle_setup` available on `fpath`
#
# ZSH Mindfuck:
#
# * ${(f)EXPR} splits value of expr by Newline
# * ${(s/ /)EXPR} splits value of expr by space, or any other char instead of space
# * ${(j/ /)EXPR) joins value of expr by space
# * ${(kv)EXPR) if EXPR is an associative array, this gives you a compacted sequence of key, value pairs.
# * ${(p...)EXPR) the p here makes the following magic to recognize Print Escapes ala '\n'
# * ${(@...)EXPR) in double quotes puts each array result into separate word
#
# ZSH Mindfuck Examples:
#
# > typeset -A K=(a b c d)
#
# > echo ${(j:.:)${(kv)K}}
# a.b.c.d
#
# > echo ${(j:.:)${(k)K}}
# a.c
#
# > echo ${(j:.:)${(v)K}}
# b.d
#
# > echo "${(j:.:)${(v)K}}"
# b d
#
# > echo "${(@j:.:)${(v)K}}"
# b d
#
# > echo "${(@j:.:)${(@v)K}}"
# b.d
#
# > echo "${(j:.:)${(@v)K}}"
# b.d
#

typeset -g C_PROMPT_NEWLINE=$'\n%{\r%}'

typeset -g G_LAST_STATUS=0
typeset -g G_PREEXEC_TS=0
typeset -g G_ASYNC_DONE=0

typeset -g G_PROMPT_DATA=""

typeset -g G_LAST_PROMPT=""

#-------------------------------------------------------------------------------

__async_prompt_query() {
  if ! (( $+commands[goprompt] )); then
    echo -n ""
  fi

  goprompt query \
    --cmd-status "$G_LAST_STATUS" \
    --preexec-ts "$G_PREEXEC_TS"
}

__async_prompt_render() {
  if ! (( $+commands[goprompt] )); then
    echo -n ""
  fi

  local MODE="normal"
  if [[ $KEYMAP == "viins" ]]; then
    MODE="edit"
  fi

  local LOADING=1
  if [[ $G_ASYNC_DONE -eq 1 ]]; then
    LOADING=0
  fi

  goprompt render \
    --prompt-mode "$MODE" \
    --prompt-loading="$LOADING" \
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

declare -A ZLE_ASYNC_FDS=()

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
  __prompt_rerender
}

__zle_async_dispatch() {
  local dispatch_handler="$1"; shift 1
  local command=( "$@" )

  # Close existing file descriptor for this handler.
  local OLD_ZLE_FD=${ZLE_ASYNC_FDS["${dispatch_handler}"]}
  if [[ -n $OLD_ZLE_FD ]]; then
    __zle_async_detach "$OLD_ZLE_FD" 2>/dev/null
  fi

  # Create File Descriptor and attach to async command
  exec {ZLE_FD}< <( "${command[@]}" )

  # Attach file a ZLE handler to file descriptor.
  zle -F $ZLE_FD "${dispatch_handler}"
  ZLE_ASYNC_FDS["${dispatch_handler}"]="$ZLE_FD"
}

__zle_async_detach() {
  local ZLE_FD=$1
  # Close stdout.
  exec {ZLE_FD}<&-
  # Close the file-descriptor.
  zle -F "$ZLE_FD"
}

__zsh_buffer_fd_reader() {
  local D=$1
  local FD=$2
  local CALLBACK=$3

  BUFFER=()
  BUFFER_NEW_DATA=0

  while; do
    local t_start=$EPOCHREALTIME

    IFS= read -t "$D" -r -u "$FD" read_result
    local read_status=$?

    local t_end=$EPOCHREALTIME

    local t_delta=$(( $t_end - $t_start ))
    if (( $t_delta < $D && $read_status != 0 )); then
      # This is EOF, since delta is less than timeout.
      # Be careful here as we might still have partial data in read_result.
      if [[ $read_result != "" ]]; then
        BUFFER+=( "$read_result" )
        BUFFER_NEW_DATA=1
      fi
      break

    elif (( $read_status == 0 )); then
      # This is buisness as usual
      BUFFER+=( "$read_result" )
      BUFFER_NEW_DATA=1

    elif [[ $BUFFER_NEW_DATA -eq 1 ]]; then
      # If we reached here we have partial results from FD.
      # But there has been a delay in fetching new data from FD.
      # Send partial results to callback.

      "${CALLBACK}" "${(pj.\n.)BUFFER}" 0
      BUFFER_NEW_DATA=0
    fi
  done

  "${CALLBACK}" "${(pj.\n.)BUFFER}" 1
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