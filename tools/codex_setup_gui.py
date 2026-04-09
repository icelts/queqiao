from __future__ import annotations

import json
import shutil
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Iterable

import tkinter as tk
from tkinter import messagebox, ttk


APP_TITLE = "QueQiao Codex 一键配置"
DEFAULT_BASE_URL = "https://queqiao.online"
PROVIDER_ID = "relay_queqiao"
PROVIDER_NAME = "QueQiao"
DEFAULT_MODEL = "gpt-5.4"
DEFAULT_REASONING_EFFORT = "xhigh"
ORIGINAL_BACKUP_NAME = "queqiao-original"
ORIGINAL_MANIFEST_NAME = "manifest.json"
MODELS = (
    "gpt-5.4",
    "gpt-5.4-pro",
    "gpt-5.4-mini",
    "gpt-5.4-nano",
    "gpt-5.3-codex",
    "gpt-5.2-codex",
    "gpt-5.1-codex",
    "gpt-5.1-codex-mini",
    "gpt-5.1-codex-max",
)


@dataclass(frozen=True)
class SetupResult:
    codex_home: Path
    config_path: Path
    auth_path: Path
    backup_dir: Path
    original_backup_dir: Path
    model: str
    base_url: str


def get_codex_home() -> Path:
    env_value = _safe_env("CODEX_HOME")
    if env_value:
        return Path(env_value).expanduser()
    return Path.home() / ".codex"


def _safe_env(key: str) -> str:
    try:
        import os
        return os.environ.get(key, "").strip()
    except Exception:
        return ""


def normalize_base_url(value: str) -> str:
    return value.strip()


def build_auth_text(api_key: str) -> str:
    return json.dumps({"OPENAI_API_KEY": api_key}, ensure_ascii=False, indent=2) + "\n"


def split_lines(content: str) -> list[str]:
    if not content:
        return []
    normalized = content.replace("\r\n", "\n").replace("\r", "\n")
    return normalized.split("\n")


def is_table_header(line: str) -> bool:
    stripped = line.strip()
    return stripped.startswith("[") and stripped.endswith("]")


def find_section_range(lines: list[str], header: str) -> tuple[int, int] | None:
    start_index: int | None = None
    for index, line in enumerate(lines):
        stripped = line.strip()
        if stripped == header:
            start_index = index
            continue
        if start_index is not None and is_table_header(stripped):
            return start_index, index
    if start_index is not None:
        return start_index, len(lines)
    return None


def insert_line(lines: list[str], index: int, value: str) -> list[str]:
    return [*lines[:index], value, *lines[index:]]


def replace_range(lines: list[str], start: int, end: int, replacement: Iterable[str]) -> list[str]:
    return [*lines[:start], *replacement, *lines[end:]]


def upsert_root_key(lines: list[str], key: str, value: str) -> list[str]:
    insert_at = 0
    for index, line in enumerate(lines):
        stripped = line.strip()
        if is_table_header(stripped):
            insert_at = index
            break
        if stripped.startswith(f"{key} ="):
            lines[index] = f"{key} = {value}"
            return lines
        insert_at = index + 1
    return insert_line(lines, insert_at, f"{key} = {value}")


def upsert_table(lines: list[str], header: str, body_lines: list[str]) -> list[str]:
    replacement = [header, *body_lines, ""]
    section_range = find_section_range(lines, header)
    if section_range is None:
        if lines and lines[-1].strip():
            lines = [*lines, ""]
        return [*lines, *replacement]
    return replace_range(lines, section_range[0], section_range[1], replacement)


def trim_trailing_blank_lines(lines: list[str]) -> list[str]:
    trimmed = list(lines)
    while trimmed and not trimmed[-1].strip():
        trimmed.pop()
    return trimmed


def merge_config(existing_text: str, model: str, base_url: str) -> str:
    lines = split_lines(existing_text)
    root_entries = (
        ("model_provider", f'"{PROVIDER_ID}"'),
        ("model", f'"{model}"'),
        ("review_model", f'"{model}"'),
        ("model_reasoning_effort", f'"{DEFAULT_REASONING_EFFORT}"'),
        ("disable_response_storage", "true"),
        ("network_access", '"enabled"'),
        ("windows_wsl_setup_acknowledged", "true"),
        ("model_context_window", "1000000"),
        ("model_auto_compact_token_limit", "900000"),
    )

    for key, value in root_entries:
        lines = upsert_root_key(lines, key, value)

    provider_lines = [
        f'name = "{PROVIDER_NAME}"',
        f'base_url = "{base_url}"',
        'wire_api = "responses"',
        "requires_openai_auth = true",
    ]
    lines = upsert_table(lines, f"[model_providers.{PROVIDER_ID}]", provider_lines)
    lines = trim_trailing_blank_lines(lines)
    return "\n".join(lines) + "\n"


def backup_file_if_exists(source: Path, destination: Path) -> None:
    if source.exists():
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, destination)


def write_original_manifest(directory: Path, config_exists: bool, auth_exists: bool) -> None:
    payload = {
        "created_at": datetime.now().isoformat(timespec="seconds"),
        "config_existed": config_exists,
        "auth_existed": auth_exists,
    }
    (directory / ORIGINAL_MANIFEST_NAME).write_text(
        json.dumps(payload, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
        newline="\n",
    )


def read_original_manifest(directory: Path) -> dict[str, object]:
    manifest_path = directory / ORIGINAL_MANIFEST_NAME
    if not manifest_path.exists():
        raise FileNotFoundError("未找到最初配置快照。")
    return json.loads(manifest_path.read_text(encoding="utf-8"))


def ensure_original_snapshot(config_path: Path, auth_path: Path, original_backup_dir: Path) -> None:
    manifest_path = original_backup_dir / ORIGINAL_MANIFEST_NAME
    if manifest_path.exists():
        return

    original_backup_dir.mkdir(parents=True, exist_ok=True)
    config_existed = config_path.exists()
    auth_existed = auth_path.exists()

    backup_file_if_exists(config_path, original_backup_dir / "config.toml")
    backup_file_if_exists(auth_path, original_backup_dir / "auth.json")
    write_original_manifest(original_backup_dir, config_exists=config_existed, auth_exists=auth_existed)


def restore_original_files(codex_home: Path | None = None) -> Path:
    codex_home = codex_home or get_codex_home()
    config_path = codex_home / "config.toml"
    auth_path = codex_home / "auth.json"
    original_backup_dir = codex_home / "backups" / ORIGINAL_BACKUP_NAME
    manifest = read_original_manifest(original_backup_dir)

    config_existed = bool(manifest.get("config_existed"))
    auth_existed = bool(manifest.get("auth_existed"))

    if config_existed:
        shutil.copy2(original_backup_dir / "config.toml", config_path)
    elif config_path.exists():
        config_path.unlink()

    if auth_existed:
        shutil.copy2(original_backup_dir / "auth.json", auth_path)
    elif auth_path.exists():
        auth_path.unlink()

    return original_backup_dir


def apply_setup(api_key: str, model: str, base_url: str, codex_home: Path | None = None) -> SetupResult:
    normalized_api_key = api_key.strip()
    normalized_base_url = normalize_base_url(base_url)

    if not normalized_api_key:
        raise ValueError("请输入网站创建的 API Key。")
    if model not in MODELS:
        raise ValueError("请选择有效的模型。")
    if not normalized_base_url:
        raise ValueError("请输入服务器地址。")

    codex_home = codex_home or get_codex_home()
    config_path = codex_home / "config.toml"
    auth_path = codex_home / "auth.json"
    backup_root = codex_home / "backups"
    backup_dir = backup_root / f"queqiao-{datetime.now().strftime('%Y%m%d-%H%M%S')}"
    original_backup_dir = backup_root / ORIGINAL_BACKUP_NAME

    codex_home.mkdir(parents=True, exist_ok=True)
    backup_dir.mkdir(parents=True, exist_ok=True)

    ensure_original_snapshot(config_path, auth_path, original_backup_dir)
    backup_file_if_exists(config_path, backup_dir / "config.toml")
    backup_file_if_exists(auth_path, backup_dir / "auth.json")

    existing_config = ""
    if config_path.exists():
        existing_config = config_path.read_text(encoding="utf-8")

    merged_config = merge_config(existing_config, model, normalized_base_url)
    config_path.write_text(merged_config, encoding="utf-8", newline="\n")
    auth_path.write_text(build_auth_text(normalized_api_key), encoding="utf-8", newline="\n")

    return SetupResult(
        codex_home=codex_home,
        config_path=config_path,
        auth_path=auth_path,
        backup_dir=backup_dir,
        original_backup_dir=original_backup_dir,
        model=model,
        base_url=normalized_base_url,
    )


def build_base_url_hint(base_url: str) -> str:
    normalized = normalize_base_url(base_url)
    if normalized.endswith("/v1"):
        return "当前服务器地址已包含 /v1。"
    return "如果配置后请求不通，可以把服务器地址改成末尾带 /v1 的形式后再重新配置。"


class CodexSetupApp:
    def __init__(self) -> None:
        self.root = tk.Tk()
        self.root.title(APP_TITLE)
        self.root.geometry("760x720")
        self.root.minsize(700, 640)
        self.root.configure(bg="#f3f6fb")
        self.root.option_add("*Font", ("Microsoft YaHei UI", 10))

        self.base_url_var = tk.StringVar(value=DEFAULT_BASE_URL)
        self.model_var = tk.StringVar(value=DEFAULT_MODEL)
        self.status_var = tk.StringVar(value="准备就绪。")
        self.base_url_hint_var = tk.StringVar(value=build_base_url_hint(DEFAULT_BASE_URL))

        self._build_ui()
        self.base_url_var.trace_add("write", self._on_base_url_changed)

    def _build_ui(self) -> None:
        style = ttk.Style(self.root)
        try:
            style.theme_use("vista")
        except tk.TclError:
            pass

        container = ttk.Frame(self.root, padding=22)
        container.pack(fill="both", expand=True)

        ttk.Label(
            container,
            text=APP_TITLE,
            font=("Microsoft YaHei UI", 18, "bold"),
        ).pack(anchor="w")

        ttk.Label(
            container,
            text="把网站创建的 API Key 粘贴进来，点击按钮后程序会自动写入用户目录 .codex 下的配置文件。",
            wraplength=660,
            foreground="#4a5b74",
            justify="left",
        ).pack(anchor="w", pady=(8, 16))

        info_frame = ttk.LabelFrame(container, text="配置目标", padding=14)
        info_frame.pack(fill="x")

        ttk.Label(
            info_frame,
            text=f"配置目录: {get_codex_home()}",
            foreground="#193457",
            wraplength=640,
            justify="left",
        ).pack(anchor="w")
        ttk.Label(
            info_frame,
            text="会自动备份当前 config.toml 和 auth.json，并保留第一次使用前的原始快照。",
            foreground="#5f6f85",
            wraplength=640,
            justify="left",
        ).pack(anchor="w", pady=(8, 0))

        status_frame = ttk.LabelFrame(container, text="状态", padding=14)
        status_frame.pack(side="bottom", fill="x", pady=(16, 0))
        ttk.Label(status_frame, textvariable=self.status_var, wraplength=640, justify="left").pack(anchor="w")

        action_frame = ttk.Frame(container)
        action_frame.pack(side="bottom", fill="x", pady=(16, 0))

        tk.Button(
            action_frame,
            text="退出",
            command=self.root.destroy,
            bg="#e5eaf3",
            fg="#1d2f4a",
            relief="flat",
            font=("Microsoft YaHei UI", 10, "bold"),
            padx=18,
            pady=10,
            cursor="hand2",
        ).pack(side="right")

        tk.Button(
            action_frame,
            text="还原最初配置",
            command=self.on_restore_original,
            bg="#f4b942",
            fg="#1d2f4a",
            activebackground="#dea431",
            activeforeground="#1d2f4a",
            relief="flat",
            font=("Microsoft YaHei UI", 10, "bold"),
            padx=18,
            pady=10,
            cursor="hand2",
        ).pack(side="right", padx=(0, 12))

        tk.Button(
            action_frame,
            text="一键配置",
            command=self.on_configure,
            bg="#1769ff",
            fg="#ffffff",
            activebackground="#0b4fd4",
            activeforeground="#ffffff",
            relief="flat",
            font=("Microsoft YaHei UI", 11, "bold"),
            padx=24,
            pady=10,
            cursor="hand2",
        ).pack(side="right", padx=(0, 12))

        form_frame = ttk.LabelFrame(container, text="填写信息", padding=14)
        form_frame.pack(fill="both", expand=True, pady=(16, 0))

        ttk.Label(form_frame, text="服务器地址").pack(anchor="w")
        ttk.Entry(form_frame, textvariable=self.base_url_var).pack(fill="x", pady=(6, 4))
        ttk.Label(
            form_frame,
            textvariable=self.base_url_hint_var,
            foreground="#5f6f85",
            wraplength=640,
            justify="left",
        ).pack(anchor="w", pady=(0, 12))

        ttk.Label(form_frame, text="模型").pack(anchor="w")
        ttk.Combobox(
            form_frame,
            textvariable=self.model_var,
            values=MODELS,
            state="readonly",
            height=len(MODELS),
        ).pack(fill="x", pady=(6, 12))

        ttk.Label(form_frame, text="API Key").pack(anchor="w")
        self.api_key_text = tk.Text(
            form_frame,
            height=8,
            wrap="word",
            undo=True,
            font=("Consolas", 11),
            relief="solid",
            borderwidth=1,
        )
        self.api_key_text.pack(fill="both", expand=True, pady=(6, 8))

        ttk.Label(
            form_frame,
            text="如果用户名、桌面路径或程序所在路径含中文，程序也会按 UTF-8 正常处理。",
            foreground="#5f6f85",
            wraplength=640,
            justify="left",
        ).pack(anchor="w")

        self.root.bind("<Control-Return>", lambda _event: self.on_configure())

    def _on_base_url_changed(self, *_args: object) -> None:
        self.base_url_hint_var.set(build_base_url_hint(self.base_url_var.get()))

    def on_configure(self) -> None:
        api_key = self.api_key_text.get("1.0", "end").strip()
        model = self.model_var.get().strip()
        base_url = self.base_url_var.get().strip()

        try:
            result = apply_setup(api_key=api_key, model=model, base_url=base_url)
        except Exception as exc:
            self.status_var.set(f"配置失败：{exc}")
            messagebox.showerror(APP_TITLE, str(exc), parent=self.root)
            return

        lines = [
            "配置成功。",
            "",
            f"服务器地址: {result.base_url}",
            f"模型: {result.model}",
            f"config.toml: {result.config_path}",
            f"auth.json: {result.auth_path}",
            f"本次备份: {result.backup_dir}",
            f"原始快照: {result.original_backup_dir}",
        ]
        if not result.base_url.endswith("/v1"):
            lines.extend(["", "如果请求不通，可以把服务器地址改成末尾带 /v1 的形式后再重新配置。"])

        success_message = "\n".join(lines)
        self.status_var.set(success_message)
        messagebox.showinfo(APP_TITLE, success_message, parent=self.root)

    def on_restore_original(self) -> None:
        should_restore = messagebox.askyesno(
            APP_TITLE,
            "这会把 .codex 恢复到第一次使用本工具之前的状态，是否继续？",
            parent=self.root,
        )
        if not should_restore:
            return

        try:
            original_backup_dir = restore_original_files()
        except Exception as exc:
            self.status_var.set(f"还原失败：{exc}")
            messagebox.showerror(APP_TITLE, str(exc), parent=self.root)
            return

        success_message = (
            "已还原到最初配置。\n\n"
            f"原始快照目录: {original_backup_dir}\n"
            "如果你已经打开了 VS Code 或 Cursor，建议重启客户端。"
        )
        self.status_var.set(success_message)
        messagebox.showinfo(APP_TITLE, success_message, parent=self.root)

    def run(self) -> None:
        self.root.mainloop()


def main() -> int:
    app = CodexSetupApp()
    app.run()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
