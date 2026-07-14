#!/usr/bin/env bash
set -euo pipefail

go test ./...
scripts/generate-llms.py --check

example_output="$(go run ./examples/e2e)"
if [[ "$example_output" != "42" ]]; then
  echo "end-to-end example returned unexpected output: $example_output" >&2
  exit 1
fi

python3 <<'PY'
import pathlib
import re
import subprocess
import sys
import urllib.parse

root = pathlib.Path.cwd()
tracked = subprocess.check_output(
    ["git", "ls-files", "-z", "*.md"],
).decode().split("\0")
link_pattern = re.compile(r"(?<!!)\[[^]]+\]\(([^)]+)\)")
failures = []

for relative in filter(None, tracked):
    document = root / relative
    text = document.read_text()
    text = re.sub(r"```.*?```", "", text, flags=re.DOTALL)
    text = re.sub(r"`[^`]*`", "", text)
    for target in link_pattern.findall(text):
        target = target.strip().strip("<>").split(" ", 1)[0]
        if target.startswith(("http://", "https://", "mailto:", "#")):
            continue
        path = urllib.parse.unquote(target.split("#", 1)[0])
        if path and not (document.parent / path).exists():
            failures.append(f"{relative}: missing Markdown link target {target}")

if failures:
    print("\n".join(failures), file=sys.stderr)
    raise SystemExit(1)

print("Markdown links and runnable examples are valid")
PY
