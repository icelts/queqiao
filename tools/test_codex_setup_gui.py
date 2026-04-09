from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path


CURRENT_DIR = Path(__file__).resolve().parent
if str(CURRENT_DIR) not in sys.path:
    sys.path.insert(0, str(CURRENT_DIR))

from codex_setup_gui import (
    DEFAULT_BASE_URL,
    ORIGINAL_BACKUP_NAME,
    PROVIDER_ID,
    apply_setup,
    build_auth_text,
    build_base_url_hint,
    merge_config,
    restore_original_files,
)


class CodexSetupGuiTests(unittest.TestCase):
    def test_merge_config_keeps_existing_sections(self) -> None:
        existing = "\n".join(
            [
                'model_provider = "openai"',
                'model = "gpt-5.1"',
                "",
                "[mcp_servers.playwright]",
                'command = "npx"',
                'args = ["@playwright/mcp@latest"]',
                "",
            ]
        )

        merged = merge_config(existing, "gpt-5.4-pro", "https://relay.example.com")

        self.assertIn('[mcp_servers.playwright]', merged)
        self.assertIn('model = "gpt-5.4-pro"', merged)
        self.assertIn(f'model_provider = "{PROVIDER_ID}"', merged)
        self.assertIn('base_url = "https://relay.example.com"', merged)

    def test_apply_setup_supports_chinese_paths(self) -> None:
        with tempfile.TemporaryDirectory(prefix="codex-setup-") as temp_dir:
            codex_home = Path(temp_dir) / "测试用户" / ".codex"
            codex_home.mkdir(parents=True, exist_ok=True)

            original_config = codex_home / "config.toml"
            original_auth = codex_home / "auth.json"
            original_config.write_text('[plugins."github@openai-curated"]\nenabled = true\n', encoding="utf-8")
            original_auth.write_text(build_auth_text("sk-old"), encoding="utf-8")

            result = apply_setup("sk-new", "gpt-5.4", DEFAULT_BASE_URL, codex_home=codex_home)

            self.assertTrue(result.config_path.exists())
            self.assertTrue(result.auth_path.exists())
            self.assertTrue(result.backup_dir.exists())
            self.assertTrue(result.original_backup_dir.exists())
            self.assertEqual(result.codex_home, codex_home)

            config_text = result.config_path.read_text(encoding="utf-8")
            auth_text = result.auth_path.read_text(encoding="utf-8")
            backup_config_text = (result.backup_dir / "config.toml").read_text(encoding="utf-8")
            backup_auth_text = (result.backup_dir / "auth.json").read_text(encoding="utf-8")

            self.assertIn(f'base_url = "{DEFAULT_BASE_URL}"', config_text)
            self.assertIn('[plugins."github@openai-curated"]', config_text)
            self.assertIn('"OPENAI_API_KEY": "sk-new"', auth_text)
            self.assertIn('enabled = true', backup_config_text)
            self.assertIn('"OPENAI_API_KEY": "sk-old"', backup_auth_text)

    def test_restore_original_files_restores_first_snapshot(self) -> None:
        with tempfile.TemporaryDirectory(prefix="codex-setup-") as temp_dir:
            codex_home = Path(temp_dir) / ".codex"
            codex_home.mkdir(parents=True, exist_ok=True)
            config_path = codex_home / "config.toml"
            auth_path = codex_home / "auth.json"

            config_path.write_text('model = "legacy"\n', encoding="utf-8")
            auth_path.write_text(build_auth_text("sk-legacy"), encoding="utf-8")

            apply_setup("sk-first", "gpt-5.4", DEFAULT_BASE_URL, codex_home=codex_home)
            apply_setup("sk-second", "gpt-5.4-pro", "https://queqiao.online/v1", codex_home=codex_home)

            original_backup_dir = restore_original_files(codex_home=codex_home)

            self.assertEqual(original_backup_dir.name, ORIGINAL_BACKUP_NAME)
            self.assertIn('model = "legacy"', config_path.read_text(encoding="utf-8"))
            self.assertIn('"OPENAI_API_KEY": "sk-legacy"', auth_path.read_text(encoding="utf-8"))

    def test_base_url_hint_mentions_v1_when_missing(self) -> None:
        self.assertIn("/v1", build_base_url_hint("https://queqiao.online"))
        self.assertIn("已包含 /v1", build_base_url_hint("https://queqiao.online/v1"))


if __name__ == "__main__":
    unittest.main()
