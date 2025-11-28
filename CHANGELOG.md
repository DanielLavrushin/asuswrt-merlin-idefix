# IDEFIX Terminal Changelog

## [1.3.0] - 2025-11-28

- ADDED: Multi-tab terminal sessions - run up to 6 independent shell sessions simultaneously.
- FIXED: TLS certificates now load from correct path (`/jffs/addons/idefix/`), resolving HTTPS connection failures.
- FIXED: Server now actually starts on router boot (missing `start` call in startup sequence).
- FIXED: TLS certificate loading errors are now logged instead of silently ignored.
- FIXED: Token expiry check now uses current token state, preventing stale reconnection attempts.
- FIXED: Update dialog now appears above terminal overlay in wide/fullscreen mode.

## [1.2.2] - 2025-05-19

- ADDED: `Idefix Terminal` now writes its status and error messages straight to the router’s System Log (tagged IDEFIX). You can follow startups, client connections, shell launches, and warnings directly from the Merlin web UI or `/tmp/syslog.log`.

## [1.2.0] - 2025-05-17

> _Important: Please clear your browser cache (e.g. **Ctrl+F5**) to ensure outdated files are updated._

- FIXED: Terminal now picks up the correct width and height on first load, window resizes, and fullscreen toggles.
- FIXED: Idefix—our super-duper mascot dog — no longer overlaps the terminal pane. If his fluffy face is still too distracting, you can hide him with a single click (cookie-based)… though I won’t ask about your conscience.

## [1.1.6] - 2025-05-11

> _Important: Please clear your browser cache (e.g. **Ctrl+F5**) to ensure outdated files are updated._

- FIXED: Resolved an issue that prevented the web-terminal from connecting over HTTPS.
- ADDED: Expand button — lets you toggle the terminal into a wider overlay. (experimental; may still show minor layout glitches).
- ADDED: The session now closes automatically when the user enters the `exit` command.

## [1.1.4] - 2025-05-09

> _Important: Please clear your browser cache (e.g. **Ctrl+F5**) to ensure outdated files are updated._

- FIXED: better update behavior.

## [1.1.2] - 2025-05-09

> _Important: Please clear your browser cache (e.g. **Ctrl+F5**) to ensure outdated files are updated._

- FIXED: visual styling.

## [1.1.1] - 2025-05-09

> _Important: Please clear your browser cache (e.g. **Ctrl+F5**) to ensure outdated files are updated._

- FIXED: update process.

## [1.0.0] - 2025-05-09

> _Important: Please clear your browser cache (e.g. **Ctrl+F5**) to ensure outdated files are updated._

- Initial Release.
