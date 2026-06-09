# hepler zsh integration.
# Add to ~/.zshrc:  source <(hepler init zsh)
# Change the key by setting HEPLER_KEY before sourcing, e.g. HEPLER_KEY='^O'.

hepler-widget() {
  emulate -L zsh
  local out
  out=$(hepler edit <<<"$BUFFER")
  if [[ $? -eq 0 && -n $out ]]; then
    BUFFER=$out
    CURSOR=${#BUFFER}
  fi
  zle reset-prompt
}
zle -N hepler-widget
bindkey "${HEPLER_KEY:-^G}" hepler-widget
