# hepler bash integration.
# Add to ~/.bashrc:  source <(hepler init bash)
# Bound to Ctrl-G; edit the bind line below to change it.

hepler-widget() {
  local out
  out=$(hepler edit <<<"$READLINE_LINE")
  if [[ $? -eq 0 && -n $out ]]; then
    READLINE_LINE=$out
    READLINE_POINT=${#READLINE_LINE}
  fi
}
bind -x '"\C-g": hepler-widget'
