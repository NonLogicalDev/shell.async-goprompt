# In a file `prompt_zle_setup` available on `fpath`:

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

typeset -gA G_PROMPT_PARTS=()

typeset -g G_LAST_PROMPT=""

#-------------------------------------------------------------------------------

__async_prompt_info() {
  if ! (( $+commands[goprompt] )); then
    echo -n "[ERROR: goprompt binary missing]"
  fi
  goprompt \
    --cmd-status "$G_LAST_STATUS" \
    --preexec-ts "$G_PREEXEC_TS"
}

#-------------------------------------------------------------------------------

__prompt_rerender() {
  local -A P=( ${(kv)G_PROMPT_PARTS} )

  local prompt_parts_top=()
  local prompt_parts_bottom=()

  if [[ ${P[vcs]} == "git" ]]; then
    local git_dirty_marks=""
    if [[ -n ${P[vcs_dirty]} && ${P[vcs_dirty]} != "0" ]]; then
      git_dirty_marks="(&)"
    fi

    local git_log_dir=""
    if [[ ${P[vcs_log_ahead]} -gt 0 || ${P[vcs_log_behind]} -gt 0 ]]; then
      git_log_dir=":[+${P[vcs_log_ahead]}:-${P[vcs_log_behind]}]"
    fi

    prompt_parts_top+=(
      "{${git_dirty_marks}git:${P[vcs_br]}${git_log_dir}}"
    )

    if [[ -n ${P[stg]} ]]; then
      local stg_dirty_marks=""
      if [[ -n ${P[stg_ditry]} && ${P[stg_dirty]} != "0" ]]; then
        stg_dirty_marks="(&)"
      fi

      local stg_patch=""
      if [[ -n ${P[stg_top]} ]]; then
        stg_patch=":${P[stg_top]}"
      fi

      local stg_pos=""
      if [[ ${P[stg_qpos]} -gt 0 || ${P[stg_qlen]} -gt 0 ]]; then
        stg_pos=":[${P[stg_qpos]:-0}/${P[stg_qlen]:-0}]"
      fi

      prompt_parts_top+=(
        "{stg${stg_patch}${stg_pos}}"
      )
    fi
  fi

  if [[ -n ${P[st]} ]]; then
    prompt_parts_bottom+=(
      "[${P[st]}]"
    )
  fi

  prompt_parts_bottom+=(
    "(${P[wd]})"
  )

  prompt_parts_bottom+=(
    "${P[ds]}"
  )

  prompt_parts_bottom+=(
    "[${P[ts]}]"
  )

  local prompt_marker="❯"
  if [[ $KEYMAP == "vicmd" ]]; then
      prompt_marker="❮"
  fi

  local prompt_prefix=":?"
  if [[ $G_ASYNC_DONE -eq 1 ]]; then
    prompt_prefix="::"
  fi

  local -a prompt_parts=()
  if [[ ${#prompt_parts_top[@]} -gt 0 ]]; then
    prompt_parts+=("${prompt_prefix} %F{yellow}${(j. .)prompt_parts_top}%f")
  else
    prompt_parts+=("${prompt_prefix} %F{yellow}-----------%f")
  fi
  if [[ ${#prompt_parts_bottom[@]} -gt 0 ]]; then
    prompt_parts+=("${prompt_prefix} %F{blue}${(j. .)prompt_parts_bottom}%f")
  else
    prompt_parts+=("${prompt_prefix} %F{blue}-----------%f")
  fi
  prompt_parts+=(
    "%F{green}$prompt_marker%f "
  )

  local BR=$C_PROMPT_NEWLINE
  PROMPT="$BR${(pj.$BR.)prompt_parts}"

  if [[ $PROMPT != $G_LAST_PROMPT ]]; then
    zle && zle .reset-prompt
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
  G_PROMPT_PARTS=()

  # set prompt status to rendering
  G_ASYNC_DONE=0

  __zle_async_dispatch __zle_async_fd_handler __async_prompt_info

  __prompt_rerender
}

#-------------------------------------------------------------------------------
# ZLE Async
#-------------------------------------------------------------------------------

declare -A ZLE_ASYNC_FDS=()

__zle_async_fd_handler() {
  local ZLE_FD=$1

  # read in all data that is available
  if ! IFS=$'\n' read -r ASYNC_RESULT <&"$ZLE_FD"; then
    # select marks this fd if we reach EOF,
    # so handle this specially.
    __zle_async_detach "$ZLE_FD"
    G_ASYNC_DONE=1
    __prompt_rerender
    return 1
  fi


  local RLINES=( "${(@f)ASYNC_RESULT}" )
  for line in "${RLINES[@]}"; do
    # split by tab char
    local KV=( "${(@s/	/)line}" )

    G_PROMPT_PARTS[${KV[1]}]=${KV[2]}
  done

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