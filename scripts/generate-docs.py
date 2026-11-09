#!/usr/bin/env python3
"""Project runnable examples, checked GoDoc APIs and measured evidence into README assets."""
import argparse
import hashlib
import html
import json
from pathlib import Path
import re
import subprocess
import textwrap

ROOT = Path(__file__).resolve().parent.parent
APIS = [
    ('New', '.', 'New', 'quickstart', 'Quick Start'),
    ('Notifier.Send', '.', 'Notifier.Send', 'quickstart', 'Quick Start'),
    ('NewMemoryPlanner', '.', 'NewMemoryPlanner', 'quickstart', 'Quick Start'),
    ('NewAdapter', '.', 'NewAdapter', 'quickstart', 'Quick Start'),
    ('WithJournal', '.', 'WithJournal', 'fallback', 'Fallback and identity'),
    ('NewMemoryJournal', '.', 'NewMemoryJournal', 'fallback', 'Fallback and identity'),
    ('webhook.New', 'driver/webhook', 'New', 'fallback', 'Fallback and identity'),
    ('notificationfake.New', 'notificationfake', 'New', 'fake', 'Application testing'),
    ('Fake.Requests', 'notificationfake', 'Fake.Requests', 'fake', 'Application testing'),
    ('notificationmail.New', 'driver/mail', 'New', 'mail', 'Mail adapter'),
]


def source_fingerprint():
    """Bind coverage evidence to Go source, tests, and module requirements."""
    digest = hashlib.sha256()
    for path in sorted(ROOT.rglob('*')):
        if path.is_file() and '.git' not in path.parts and '.artifacts' not in path.parts and (path.suffix == '.go' or path.name in ('go.mod', 'go.sum')):
            digest.update(str(path.relative_to(ROOT)).encode() + b'\0' + path.read_bytes())
    return digest.hexdigest()


def snippet(source):
    """Extract exactly one marked region from an authoritative runnable program."""
    if source.count('// snippet:start') != 1 or source.count('// snippet:end') != 1:
        raise ValueError('expected exactly one snippet region')
    region = source.split('// snippet:start', 1)[1].split('// snippet:end', 1)[0]
    if 'fmt.Println(' not in region or not re.search(r'\n\s*// [^\n]+', region):
        raise ValueError('snippet needs adjacent expected output')
    return textwrap.dedent(region).strip()


def badge(label, value, color='#6078d8', icon=False):
    """Render a repository-owned accessible SVG with an optional bell icon."""
    text = label if not value else label + ': ' + value
    width = max(100, 30 + len(text) * 7)
    shape = '<path d="M9 17h10l-2-3v-4a3 3 0 0 0-6 0v4l-2 3m4 3h2" fill="none" stroke="white" stroke-width="1.5"/>' if icon else ''
    return f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="28" role="img" aria-label="{html.escape(text)}"><title>{html.escape(text)}</title><rect width="{width}" height="28" rx="5" fill="{color}"/>{shape}<text x="{28 if icon else 12}" y="18" fill="white" font-family="Verdana,sans-serif" font-size="12">{html.escape(text)}</text></svg>\n'


def outputs():
    """Validate inputs before producing the complete deterministic projection."""
    evidence = json.loads((ROOT / 'docs/unit-results.json').read_text())
    if evidence.get('source_fingerprint') != source_fingerprint():
        raise ValueError('unit evidence is stale; run make test')
    if not evidence['total_statements'] or evidence['covered_statements'] != evidence['total_statements']:
        raise ValueError('full library unit coverage is required')
    snippets = {p.parent.name: snippet(p.read_text()) for p in sorted((ROOT / 'examples').glob('*/main.go'))}
    readme = (ROOT / 'docs/readme.tmpl.md').read_text()
    for name, code in snippets.items():
        marker = '{{snippet:' + name + '}}'
        if readme.count(marker) != 1:
            raise ValueError(f'example {name} needs one README target')
        readme = readme.replace(marker, '```go\n' + code + '\n```')
    rows = ['| API | Example |', '| --- | --- |']
    for label, package, symbol, example, heading in APIS:
        subprocess.run(['go', 'doc', '.', symbol], cwd=ROOT / package, check=True, capture_output=True)
        operation = symbol.split('.')[-1]
        if not re.search(r'\.' + re.escape(operation) + r'\(', snippets[example]):
            raise ValueError(f'{label} is absent from its linked example')
        anchor = heading.lower().replace(' ', '-')
        if f'## {heading}\n' not in readme:
            raise ValueError(f'missing anchor {anchor}')
        rows.append(f'| [{label}](#{anchor}) | {heading} |')
    readme = readme.replace('{{api-index}}', '\n'.join(rows))
    if re.search(r'\{\{(?:snippet:|api-index)', readme):
        raise ValueError('unresolved README marker')
    result = {'README.md': readme}
    for name, label, value in [('status', 'status', 'unreleased'), ('go', 'Go', '1.24+'), ('license', 'license', 'MIT'), ('coverage', 'library unit coverage', '100%'), ('tests', 'unit tests', str(evidence['unit_tests_passed']))]:
        result[f'docs/assets/badges/{name}.svg'] = badge(label, value)
    for name, label in [('intent', 'Semantic intent'), ('policy', 'Recipient policy'), ('channels', 'Typed channels'), ('identity', 'Explicit identity'), ('diagnostics', 'Safe diagnostics'), ('testing', 'Isolated testing')]:
        result[f'docs/assets/badges/{name}.svg'] = badge(label, '', '#33436b', True)
    return result


def main():
    """Check or update all generated files without changing authoritative examples."""
    parser = argparse.ArgumentParser()
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    for name, content in outputs().items():
        path = ROOT / name
        if args.check:
            if not path.exists() or path.read_text() != content:
                raise SystemExit(f'generated file is stale: {name}')
        else:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content)


if __name__ == '__main__':
    main()
