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
# b d #
# > echo "${(@j:.:)${(v)K}}"
# b d
#
# > echo "${(@j:.:)${(@v)K}}"
# b.d
#
# > echo "${(j:.:)${(@v)K}}"
# b.d
#

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


__prompt_rerender() {
  local -A P=( )

  local RLINES=( "${(@f)G_PROMPT_DATA}" )
  for line in "${RLINES[@]}"; do
    # split by tab char
    local KV=( "${(@s/	/)line}" )

    if [[ ${#KV[@]} -ge 2 ]]; then
      P[${KV[1]}]=${KV[2]}
    fi
  done

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


__zle_async_fd_handler() {
  local ZLE_FD=$1

  __callback() {
    G_PROMPT_DATA="$1"
    G_ASYNC_DONE="$2"

    if [[ $G_ASYNC_DONE -eq 1 ]]; then
      __zle_async_detach "$ZLE_FD"
    fi

    __prompt_rerender
  }

  __zsh_buffer_fd_reader 0.01 $ZLE_FD __callback
  __prompt_rerender
}


(
	exec {TEST_FDA}< <(
		sleep 1;
		echo "line 1 is long";
		echo "And also a few more strings"
		pwd
		ls /
		sleep 2;
		echo "line 2 is also long";
		sleep 3;
		ls;
		printf "DONE"
		printf ";"
		printf "\n"
		printf "Really Done Now"
	)

	__callback() {
	  echo ">>>>>>>>>>>>>>>> $1"
	}

  __zsh_buffer_fd_reader "0.05" $TEST_FDA __callback
)