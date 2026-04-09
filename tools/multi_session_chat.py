#!/usr/bin/env python3
"""
Run a few independent multi-turn chat sessions against Sub2API.

Usage:
  python tools/multi_session_chat.py
"""

from __future__ import annotations

import json
import sys
import urllib.error
import urllib.request
from typing import Any, Dict, List, Tuple

try:
    from curl_cffi import requests as curl_requests
except ImportError:  # pragma: no cover - fallback for environments without curl_cffi
    curl_requests = None


# Edit these values directly.
BASE_URL = "https://queqiao.online".rstrip("/")
API_KEY = "这里替换成你的key".strip()
MODEL = "gpt-5.4".strip()
IMPERSONATE = "chrome".strip()


def post_json(path: str, payload: Dict[str, Any]) -> Tuple[int, str]:
    url = f"{BASE_URL}{path}"
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {API_KEY}",
        "Referer": f"{BASE_URL}/dashboard",
        "User-Agent": (
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
            "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"
        ),
    }

    if curl_requests is not None:
        session = curl_requests.Session(impersonate=IMPERSONATE)
        resp = session.post(url, headers=headers, json=payload, timeout=120)
        return resp.status_code, resp.text

    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(url=url, data=data, method="POST", headers=headers)

    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            return resp.status, resp.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read().decode("utf-8")


def chat(messages: List[Dict[str, str]]) -> Dict[str, Any]:
    status, body = post_json(
        "/v1/chat/completions",
        {
            "model": MODEL,
            "messages": messages,
            "temperature": 0,
            "max_tokens": 32,
            "stream": False,
        },
    )
    if status != 200:
        raise RuntimeError(f"chat request failed: status={status} body={body}")

    return json.loads(body)


def run_session(label: str) -> Dict[str, Any]:
    messages: List[Dict[str, str]] = [
        {"role": "system", "content": f"You are session {label}."},
        {
            "role": "user",
            "content": f"Turn 1 for {label}: reply with exactly {label}-1",
        },
    ]

    first = chat(messages)
    first_text = first["choices"][0]["message"]["content"].strip()
    messages.append({"role": "assistant", "content": first_text})
    messages.append(
        {
            "role": "user",
            "content": f"Turn 2 for {label}: reply with exactly {label}-2 and refer to the previous answer.",
        }
    )

    second = chat(messages)
    second_text = second["choices"][0]["message"]["content"].strip()

    return {
        "label": label,
        "turn1": first_text,
        "turn2": second_text,
        "usage1": first.get("usage", {}),
        "usage2": second.get("usage", {}),
    }


def main() -> int:
    if not API_KEY or API_KEY == "PASTE_YOUR_API_KEY_HERE":
        print("Edit tools/multi_session_chat.py and set API_KEY first", file=sys.stderr)
        return 2

    print(f"base_url={BASE_URL}")
    print(f"model={MODEL}")

    sessions = ["A", "B", "C"]
    results = []
    for label in sessions:
        result = run_session(label)
        results.append(result)
        print(f"[session {label}] turn1={result['turn1']}")
        print(f"[session {label}] turn2={result['turn2']}")
        print(f"[session {label}] usage1={result['usage1']}")
        print(f"[session {label}] usage2={result['usage2']}")

    print(json.dumps({"sessions": results}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
