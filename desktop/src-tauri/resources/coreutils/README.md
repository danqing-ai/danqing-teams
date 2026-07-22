# Bundled Microsoft Coreutils for Windows

`coreutils.exe` is downloaded by `scripts/fetch_windows_coreutils.sh` during
`make pack-windows-desktop` (not committed). At runtime the desktop shell copies
it to `~/.dq-teams/bin/coreutils/` and creates applet hardlinks for `exec_shell`.

See https://github.com/microsoft/coreutils (MIT).
