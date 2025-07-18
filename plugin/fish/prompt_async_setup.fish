set --global _fish_async_prompt_exec {$GOPROMPT}

# ------------------------------------------------------------------------------

# Utility function to repaint the prompt
function _fish_async_prompt_repaint 
    commandline -f repaint 2>/dev/null >/dev/null
end

# Utility function to kill a pid safely
function _fish_async_prompt_kill_pid_safely -a pid_to_kill
    if kill -0 $pid_to_kill >/dev/null 2>/dev/null
        kill -9 $pid_to_kill >/dev/null 2>/dev/null
    end
end

# Utility function to check if an executable is available
function _fish_async_prompt_check_exec -a exec_name
    if not type -q $exec_name
        return 1
    end
    return 0
end

# ------------------------------------------------------------------------------

set --global _fish_async_prompt_state_var_name _fish_async_prompt_state_var_$fish_pid
set --global _fish_async_prompt_state_job_pid ""
set --global _fish_async_last_cmd_status 0
set --global _fish_async_last_cmd_preexec_ts_epoch 0

function _fish_async_prompt_kill_async_job
    if test -n "$_fish_async_prompt_state_job_pid"
        _fish_async_prompt_kill_pid_safely $_fish_async_prompt_state_job_pid
    end
    set --global _fish_async_prompt_state_job_pid ""
end

# HOOK: Erase the state variable on exit
function _fish_async_prompt_evt_fish_exit --on-event fish_exit
    _fish_async_prompt_kill_async_job
    set --erase $_fish_async_prompt_state_var_name
end

# ------------------------------------------------------------------------------

# HOOK: Repaint prompt on variable change
function _fish_async_prompt_evt_var_change_state --on-variable $_fish_async_prompt_state_var_name
    _fish_async_prompt_repaint
end

# ------------------------------------------------------------------------------

set --global _fish_async_prompt_script '
# Fire off "goprompt query" command and read the output line by line
set --local query_output

$_FISH_ASYNC_PROMPT_EXEC query \
    --cmd-status="$_FISH_ASYNC_PROMPT_LAST_CMD_STATUS" \
    --preexec-ts="$_FISH_ASYNC_PROMPT_LAST_CMD_PREEXEC_TS_EPOCH" \
    --pid-parent-skip=1 \
| while read -l line
    # Append line to query_output array
    set -a query_output $line

    # If the line is empty
    if test -z "$line"
        # Join lines together with newline and set the _async_fish_state_var_name
        set -U "$_FISH_ASYNC_PROMPT_STATE_VAR_REF" "$(string join "<%ab@xv%>" $query_output | sed "s/<%ab@xv%>/\\n/g")"
    end
end
'

# Our pretend async work function
function _fish_async_prompt_start_async_work 
    _fish_async_prompt_kill_async_job

    # Run the loop in a private subshell
    env \
        _FISH_ASYNC_PROMPT_EXEC=$_fish_async_prompt_exec \
        _FISH_ASYNC_PROMPT_STATE_VAR_REF=$_fish_async_prompt_state_var_name \
        _FISH_ASYNC_PROMPT_LAST_CMD_STATUS=$_fish_async_last_cmd_status \
        _FISH_ASYNC_PROMPT_LAST_CMD_PREEXEC_TS_EPOCH=$_fish_async_last_cmd_preexec_ts_epoch \
        fish --private --command "$_fish_async_prompt_script" &
    set --global _fish_async_prompt_state_job_pid $last_pid
    disown $last_pid
end

# ------------------------------------------------------------------------------

# Run _async_work in foreground when the prompt is requested first
function _fish_async_prompt_update --on-event fish_prompt
    _fish_async_prompt_start_async_work
end

function _fish_async_prompt_update_last_cmd_status --on-event fish_preexec
    set --global _fish_async_last_cmd_status $status
    set --global _fish_async_last_cmd_preexec_ts_epoch (date +%s)
end

# ------------------------------------------------------------------------------

# Main prompt function
function fish_prompt
    set --local state_contents $$_fish_async_prompt_state_var_name

    printf "%s " "$(printf "%s" $state_contents | $_fish_async_prompt_exec render --escape-mode ascii)"
end