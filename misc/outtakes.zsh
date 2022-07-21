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