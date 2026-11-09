#!/usr/bin/env python3
"""Check every Go module is inventoried and intentional local replacements remain."""
from pathlib import Path

root = Path(__file__).resolve().parent.parent
expected = set((root / 'scripts/module-manifest.txt').read_text().splitlines())
actual = {str(path.parent.relative_to(root)) for path in root.rglob('go.mod') if '.artifacts' not in path.parts}
if actual != expected:
    raise SystemExit(f'module manifest mismatch: {actual ^ expected}')
for name in ('driver/mail', 'examples/mail'):
    if 'replace github.com/goforj/notification => ../..' not in (root / name / 'go.mod').read_text():
        raise SystemExit(f'{name}: unpublished root replacement missing')
print('Module inventory and local root replacements verified')
