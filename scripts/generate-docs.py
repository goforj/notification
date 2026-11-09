#!/usr/bin/env python3
"""Project runnable examples, checked GoDoc APIs and measured evidence into README assets."""
import argparse
import base64
import hashlib
import html
import json
from pathlib import Path
import re
import subprocess
import textwrap
import xml.etree.ElementTree as ET

ROOT = Path(__file__).resolve().parent.parent
APIS = [
    ('New', '.', 'New', 'quickstart', 'Quick Start'),
    ('Fanout', '.', 'Fanout', 'composition', 'Composition'),
    ('Fallback', '.', 'Fallback', 'composition', 'Composition'),
    ('DeliveryError.RetryAfter', '.', 'DeliveryError.RetryAfter', 'composition', 'Composition'),
    ('pagerduty.Driver.Trigger', 'driver/pagerduty', 'Driver.Trigger', 'incident', 'Incident lifecycle'),
    ('pagerduty.Driver.Acknowledge', 'driver/pagerduty', 'Driver.Acknowledge', 'incident', 'Incident lifecycle'),
    ('pagerduty.Driver.Resolve', 'driver/pagerduty', 'Driver.Resolve', 'incident', 'Incident lifecycle'),
    ('Notifier.Send', '.', 'Notifier.Send', 'quickstart', 'Quick Start'),
    ('webhook.New', 'driver/webhook', 'New', 'webhook', 'Webhook delivery'),
    ('notificationfake.New', 'notificationfake', 'New', 'fake', 'Application testing'),
    ('Fake.Messages', 'notificationfake', 'Fake.Messages', 'fake', 'Application testing'),
    ('notificationmail.New', 'driver/mail', 'New', 'mail', 'Mail delivery'),
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


def badge(label, value, color='#007ec6'):
    """Render compact status segments from measured repository evidence."""
    title = html.escape(label + (': ' + value if value else ''))
    left = text_width(label) + 12
    right = text_width(value) + 12
    width = left + right
    return f'''<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="20" role="img" aria-label="{title}"><title>{title}</title><clipPath id="badge"><rect width="{width}" height="20" rx="3"/></clipPath><g clip-path="url(#badge)"><path fill="#555" d="M0 0h{left}v20H0z"/><path fill="{color}" d="M{left} 0h{right}v20H{left}z"/></g><g fill="#fff" text-anchor="middle" font-family="Verdana,DejaVu Sans,sans-serif" font-size="11"><text x="{left / 2:g}" y="14">{html.escape(label)}</text><text x="{left + right / 2:g}" y="14">{html.escape(value)}</text></g></svg>\n'''


def artwork_badge(presentation, directory):
    """Embed driver and feature artwork with shared dimensions, spacing and colors."""
    path = ROOT / 'docs/assets' / directory / presentation['icon']
    label = html.escape(presentation['label'])
    accent = presentation.get('accent', '#576574')
    channels = [int(accent[index:index+2], 16) for index in (1, 3, 5)]
    background = '#' + ''.join(f'{round(channel * 0.08 + 255 * 0.92):02x}' for channel in channels)
    border = '#' + ''.join(f'{round(channel * 0.28 + 255 * 0.72):02x}' for channel in channels)
    if path.suffix == '.svg':
        ET.register_namespace('', 'http://www.w3.org/2000/svg')
        icon = ET.fromstring(path.read_text())
        for parent in icon.iter():
            for child in list(parent):
                if child.tag.endswith('foreignObject'):
                    parent.remove(child)
        icon.attrib.update(x='3', y='3', width='16', height='16')
        if 'fill' in presentation:
            icon.set('fill', presentation['fill'])
        artwork = ET.tostring(icon, encoding='unicode')
    else:
        encoded = base64.b64encode(path.read_bytes()).decode('ascii')
        artwork = f'<image x="3" y="3" width="16" height="16" href="data:image/png;base64,{encoded}"/>'
    return f'''<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" role="img" aria-label="{label}"><title>{label}</title><rect x=".5" y=".5" width="21" height="21" rx="4" fill="{background}" stroke="{border}"/>{artwork}</svg>\n'''


def outputs():
    """Validate inputs before producing the complete deterministic projection."""
    evidence = json.loads((ROOT / 'docs/unit-results.json').read_text())
    if evidence.get('source_fingerprint') != source_fingerprint():
        raise ValueError('unit evidence is stale; run make test')
    if not evidence['total_statements'] or evidence['covered_statements'] != evidence['total_statements']:
        raise ValueError('full library unit coverage is required')
    snippets = {p.parent.name: snippet(p.read_text()) for p in sorted((ROOT / 'examples').glob('*/main.go'))}
    templates = {'README.md': (ROOT / 'docs/readme.tmpl.md').read_text()}
    advanced = ROOT / 'docs/advanced.tmpl.md'
    if advanced.exists():
        templates['docs/advanced.md'] = advanced.read_text()
    for name, code in snippets.items():
        marker = '{{snippet:' + name + '}}'
        if sum(content.count(marker) for content in templates.values()) != 1:
            raise ValueError(f'example {name} needs one documentation target')
        templates = {path: content.replace(marker, '```go\n' + code + '\n```') for path, content in templates.items()}
    catalog = json.loads((ROOT / 'docs/providers/drivers.json').read_text())['drivers']
    if any(entry['verification'].startswith('Real ') for entry in catalog):
        integration = json.loads((ROOT / 'docs/integration-results.json').read_text())
        versions = json.loads((ROOT / 'integration/versions.json').read_text())
        if integration['source_fingerprint'] != source_fingerprint() or integration['servers'] != versions or not {'TestNtfy', 'TestGotify'} <= set(integration['passed_tests']):
            raise ValueError('native integration evidence is stale; run make test-integration')
    presentations = json.loads((ROOT / 'docs/assets/driver-icons/presentation.json').read_text())
    driver_assets = {}
    driver_rows = ['| Driver | Configuration | Successful handling | Verification |', '| ---: | --- | --- | --- |']
    for entry in catalog:
        if not (ROOT / entry['package']).is_dir():
            raise ValueError('missing driver package: ' + entry['package'])
        presentation = presentations[entry['id']]
        asset = f"docs/assets/badges/driver-{entry['id']}.svg"
        driver_assets[asset] = artwork_badge(presentation, 'driver-icons')
        driver_rows.append(f'''| [<img src="{asset}" width="22" height="22" alt="" align="middle"> {presentation['label']}]({entry['package']}) | {entry['configuration']} | {entry['acceptance']} | {entry['verification']} |''')
    templates['README.md'] = templates['README.md'].replace('{{drivers}}', '\n'.join(driver_rows))
    readme = templates['README.md']
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
    templates['README.md'] = readme
    if any(re.search(r'\{\{(?:snippet:|api-index)', content) for content in templates.values()):
        raise ValueError('unresolved README marker')
    result = templates
    result.update(driver_assets)
    for name, label, value, color in [('status', 'status', 'unreleased', '#6e7681'), ('go', 'Go', '1.24+', '#007d9c'), ('license', 'license', 'MIT', '#007ec6'), ('coverage', 'coverage', '100%', '#2e8540'), ('tests', 'tests', str(evidence['unit_tests_passed']), '#5368bd')]:
        result[f'docs/assets/badges/{name}.svg'] = badge(label, value, color)
    features = json.loads((ROOT / 'docs/assets/feature-icons/presentation.json').read_text())
    for name, presentation in features.items():
        result[f'docs/assets/badges/{name}.svg'] = artwork_badge(presentation, 'feature-icons')
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
