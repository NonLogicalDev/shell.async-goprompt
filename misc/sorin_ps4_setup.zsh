
__goprompt_setup_handler() {
  # Setup Debug Prompt
  ################################################################################

  # Store prompt expansion symbols for in-place expansion via (%). For
  # some reason it does not work without storing them in a variable first.
  typeset -ga prompt_pure_debug_depth
  prompt_pure_debug_depth=('%e' '%N' '%x')

  # Compare is used to check if %N equals %x. When they differ, the main
  # prompt is used to allow displaying both file name and function. When
  # they match, we use the secondary prompt to avoid displaying duplicate
  # information.
  local -A ps4_parts
  ps4_parts=(
      depth 	  '%F{yellow}${(l:${(%)prompt_pure_debug_depth[1]}::+:)}%f'
      compare   '${${(%)prompt_pure_debug_depth[2]}:#${(%)prompt_pure_debug_depth[3]}}'
      main      '%F{blue}${${(%)prompt_pure_debug_depth[3]}:t}%f%F{242}:%I%f %F{242}@%f%F{blue}%N%f%F{242}:%i%f'
      secondary '%F{blue}%N%f%F{242}:%i'
      prompt 	  '%F{242}>%f '
  )
  # Combine the parts with conditional logic. First the `:+` operator is
  # used to replace `compare` either with `main` or an empty string. Then
  # the `:-` operator is used so that if `compare` becomes an empty
  # string, it is replaced with `secondary`.
  local ps4_symbols='${${'${ps4_parts[compare]}':+"'${ps4_parts[main]}'"}:-"'${ps4_parts[secondary]}'"}'

  # Improve the debug prompt (PS4), show depth by repeating the +-sign and
  # add colors to highlight essential parts like file and function name.
  PROMPT4="${ps4_parts[depth]} ${ps4_symbols}${ps4_parts[prompt]}"

  ################################################################################
}
