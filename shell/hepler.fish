# hepler fish integration.
# Add to ~/.config/fish/config.fish:  hepler init fish | source
# Bound to Ctrl-G; edit the bind line below to change it.

function hepler-widget
    # string collect keeps multi-line output as a single value.
    set -l out (commandline | hepler edit | string collect)
    if test $status -eq 0 -a -n "$out"
        commandline -r -- $out
    end
end
bind \cg hepler-widget
