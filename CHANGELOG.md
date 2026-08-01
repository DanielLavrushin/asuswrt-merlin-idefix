# IDEFIX Terminal Changelog

## [1.5.0] - 2026-08-01

> _Important: this release fixes a security issue in how the terminal's signing key was logged — see the SECURITY notes below. Please also clear your browser cache (e.g. **Ctrl+F5**), as the web UI changed substantially._

- FIXED: **Safari disconnected the terminal constantly** ([#11](https://github.com/DanielLavrushin/asuswrt-merlin-idefix/issues/11)). Safari freezes a page into its back/forward cache when you switch tabs or navigate elsewhere in the router UI, and closes the WebSocket with `WebSocket is closed due to suspension.` — leaving a dead terminal behind an unresponsive overlay. Idefix now notices the page coming back and reconnects on its own. Chrome was unaffected because it doesn't cache pages that hold an open WebSocket.
- ADDED: **Shell sessions survive a disconnect.** Your shell stays alive for 2 minutes after the browser drops off, and reconnecting picks it up where you left it — same shell, same working directory, same variables, with recent output replayed into the terminal. Switching tabs or briefly losing WiFi no longer costs you your session.
- FIXED: The terminal now reconnects automatically after _any_ dropped connection, retrying with a backoff, instead of only doing so when its token happened to be stale. The manual Reconnect button appears only once automatic retries are exhausted.
- FIXED: Pasting more than 32 KB into the terminal killed the connection. The limit is now 1 MB.
- FIXED: Closing a terminal tab now shuts down its shell immediately instead of leaving it running until it times out.
- SECURITY: The terminal's HMAC signing key was written to the router's system log every time a token was generated — **including with debug logging turned off**, because the log level was only applied to console output, not to the syslog entry. Anyone holding that key can open a root shell on the terminal port without logging into the router UI. If you have ever shared a system log from a router running Idefix — a forum thread, a bug report — treat that key as public. It is replaced automatically the next time the addon starts, which happens on every reboot and on every update, so no manual action is required.
- SECURITY: `sec.key` and the token file are now created unreadable to other users, rather than written world-readable and tightened afterwards. The token file was previously mode 664.
- SECURITY: A terminal session can now only be resumed by the browser that opened it; a session identifier is no longer sufficient on its own.
- SECURITY: Tokens dated in the future are now rejected. Previously a token minted while the router's clock ran ahead of NTP would never expire.
- SECURITY: The client identifier supplied by the browser is now validated before it is signed and written into `token.json`.

## [1.4.5] - 2026-04-13

- FIXED: Installation broken on firmware 3006+ by ASD (Asuswrt Signature Detection) quarantining `/jffs/scripts/idefix` to `/jffs/.asdbk`. The addon script is now placed at `/jffs/addons/idefix/idefix.sh`, which ASD does not scan. Hook entries in `/jffs/scripts/post-mount` and `/jffs/scripts/service-event` are rewritten to point at the new location, and the legacy `/jffs/scripts/idefix` file plus any stale `#idefix` hook lines are cleaned up automatically on upgrade.
- FIXED: Terminal unreachable from VPN clients (OpenVPN/WireGuard) after `idefix restart`.

## [1.4.2] - 2026-04-12

- ADDED: **Quick Commands palette** — press `Ctrl+K` or click the command button in the tab bar to open a searchable list of common router commands (logs, network, WiFi, VPN, firewall, NVRAM, Entware, and more). Commands are organized by category and can be filtered by typing.
- ADDED: **Session export** — download the terminal scrollback as a `.log` file using the save button in the terminal toolbar. Useful for sharing debug output on forums.
- ADDED: **Clear terminal** button in the tab bar to clear the active terminal's scrollback.
- FIXED: System Log timestamps from Idefix now match the router's local time.
- FIXED: Update certificate loading logic to support multiple paths.
- FIXED: UI stopped loading on some Gnuton and third-party firmware builds. The router's httpd exports `LD_LIBRARY_PATH` pointing to firmware libraries; Entware binaries (jq, xray, find, grep etc.) inherited this and loaded an incompatible libc, causing segfaults.

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
