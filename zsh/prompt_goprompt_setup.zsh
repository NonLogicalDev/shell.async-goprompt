# In a file `prompt_goprompt_setup` available on `fpath`:

typeset -g GOPROMPT_NEWLINE=$'\n%{\r%}'

typeset -g GOPROMPT_LAST_STATUS=0
typeset -g GOPROMPT_PREEXEC_TS=0

typeset -gA GOPROMPT_PARTS=()

#-------------------------------------------------------------------------------

autoload -Uz add-zsh-hook

#-------------------------------------------------------------------------------

__goprompt_update_handler() {
  local BR=$GOPROMPT_NEWLINE
  local -A P=( "${(@kv)GOPROMPT_PARTS}" )

  local GOPROMPT_PARTS_BOTTOM=()
  local GOPROMPT_PARTS_TOP=()

  local -a prompt_parts=(
    ":: (git: ${P[git_st]})"
    ":: ${P[wd]}"
    "# "
  )

  PROMPT="$BR${(pj.$BR.)prompt_parts}"
  zle && zle reset-prompt
}

__goprompt_async_handler() {
  local ZLE_FD=$1

  if ! IFS=$'\n' read -r ASYNC_RESULT <&"$ZLE_FD"; then
    # select marks this fd if we reach EOF,
    # so handle this specially.
    __zle_async_detach "$ZLE_FD"
    return 1
  fi

  # split by tab char
  local KV=( "${(@s/	/)ASYNC_RESULT}" )

  GOPROMPT_PARTS[${KV[1]}]=${KV[2]}

  __goprompt_update_handler
}

__goprompt_preexec() {
    typeset -g GOPROMPT_PREEXEC_TS=$EPOCHSECONDS
}

__goprompt_precmd() {
  GOPROMPT_LAST_STATUS=$?
  GOPROMPT_PARTS=()

  __goprompt_update_handler
  __zle_async_dispatch __goprompt_async_handler \
    __goprompt_run query \
      --cmd-status "$GOPROMPT_LAST_STATUS" \
      --preexec-ts "$GOPROMPT_PREEXEC_TS"
}

#-------------------------------------------------------------------------------
# ZLE Async
#-------------------------------------------------------------------------------

declare -A ZLE_ASYNC_FDS=()

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

#-------------------------------------------------------------------------------

__goprompt_run() {
  if ! (( $+commands[goprompt] )); then
    echo -n "[ERROR: goprompt binary missing]"
  fi
  goprompt "$@"
}


prompt_goprompt_setup() {
  add-zsh-hook precmd  __goprompt_precmd
  add-zsh-hook preexec __goprompt_preexec
}

prompt_goprompt_setup "$@"