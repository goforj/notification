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


def text_width(text):
    """Estimate proportional badge text widths at the shared 11px font size."""
    return sum(3 if c in "ilI.,:;!|' " else 9 if c in "mwMW@%" else 7 if c.isupper() or c.isdigit() else 6 for c in text)


def badge(label, value, color='#007ec6', icon=False):
    """Render compact status segments or a single icon and feature label."""
    title = html.escape(label + (': ' + value if value else ''))
    if icon:
        width = text_width(label) + 32
        return f'''<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="22" role="img" aria-label="{title}"><title>{title}</title><rect width="{width}" height="22" rx="3" fill="{color}"/><path d="M7 14h9l-2-3V8a2.5 2.5 0 0 0-5 0v3l-2 3m3 3h3" fill="none" stroke="white" stroke-width="1.3"/><text x="23" y="15" fill="white" font-family="Verdana,DejaVu Sans,sans-serif" font-size="11">{html.escape(label)}</text></svg>\n'''
    left = text_width(label) + 12
    right = text_width(value) + 12
    width = left + right
    return f'''<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="20" role="img" aria-label="{title}"><title>{title}</title><clipPath id="badge"><rect width="{width}" height="20" rx="3"/></clipPath><g clip-path="url(#badge)"><path fill="#555" d="M0 0h{left}v20H0z"/><path fill="{color}" d="M{left} 0h{right}v20H{left}z"/></g><g fill="#fff" text-anchor="middle" font-family="Verdana,DejaVu Sans,sans-serif" font-size="11"><text x="{left / 2:g}" y="14">{html.escape(label)}</text><text x="{left + right / 2:g}" y="14">{html.escape(value)}</text></g></svg>\n'''


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
    for name, label, value, color in [('status', 'status', 'unreleased', '#6e7681'), ('go', 'Go', '1.24+', '#007d9c'), ('license', 'license', 'MIT', '#007ec6'), ('coverage', 'coverage', '100%', '#2e8540'), ('tests', 'tests', str(evidence['unit_tests_passed']), '#5368bd')]:
        result[f'docs/assets/badges/{name}.svg'] = badge(label, value, color)
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
