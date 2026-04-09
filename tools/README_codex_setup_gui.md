# QueQiao Codex Setup GUI

This Windows GUI tool lets users paste an API key, choose a model, edit the server address, and configure Codex in one click.

What it writes:

- `%USERPROFILE%\.codex\config.toml`
- `%USERPROFILE%\.codex\auth.json`

What it preserves:

- existing non-conflicting sections in `config.toml`
- backups of old `config.toml` and `auth.json` under `%USERPROFILE%\.codex\backups\`
- the very first pre-tool snapshot, so users can restore their original Codex config later

Default server address:

```text
https://queqiao.online
```

If requests do not work with that address, the GUI reminds users they can try appending `/v1`.

Models included in the dropdown:

- `gpt-5.4`
- `gpt-5.4-pro`
- `gpt-5.4-mini`
- `gpt-5.4-nano`
- `gpt-5.3-codex`
- `gpt-5.2-codex`
- `gpt-5.1-codex`
- `gpt-5.1-codex-mini`
- `gpt-5.1-codex-max`

## Run locally

```powershell
python .\tools\codex_setup_gui.py
```

## Run tests

```powershell
python -m unittest .\tools\test_codex_setup_gui.py
```

## Build Windows exe

```powershell
powershell -ExecutionPolicy Bypass -File .\tools\build_codex_setup_exe.ps1
```

Output path:

```text
dist\queqiao-codex-setup.exe
```
