#!/usr/bin/env python3
"""Snapshot Uptime Gopher provider IDs and distinguish native adapters from aliases."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parent.parent


def inventory(checkout):
    """Read authoritative demo templates without rendering or changing the framework."""
    directory = checkout / 'templates/demo/internal/notification'
    source_paths = {
        'constants': directory / 'channel_repo.go.tmpl',
        'catalog': directory / 'provider_catalog.go.tmpl',
        'manager': directory / 'manager.go.tmpl',
        'generic': directory / 'generic_webhook_compat_provider.go.tmpl',
        'ui': checkout / 'templates/demo/frontend/src/lib/notification-providers.ts',
    }
    sources = {name: path.read_text() for name, path in source_paths.items()}
    constants = dict(re.findall(r'(Provider\w+)\s*=\s*"([^\"]+)"', sources['constants']))

    def provider(token):
        return constants.get(token.strip(), token.strip().strip('"'))

    catalog = sources['catalog'].split('func NormalizeProviderID')[0]
    ids = {provider(token) for token in re.findall(r'^\s*("[^\"]+"|Provider\w+):\s*\{\}', catalog, re.M)}
    special = sources['catalog'].split('func IsSupportedNotificationProvider', 1)[1].split('func IsWebhookCompatProvider', 1)[0]
    ids.update(provider(token) for token in re.findall(r'normalized == (Provider\w+)', special))
    ui_array = re.search(r'const webhookCompatProviderIDs[^=]*=\s*\[(.*?)\]', sources['ui'], re.S)
    ui = set(re.findall(r"'([^']+)'", ui_array.group(1))) | set(re.findall(r"\bid:\s*'([^']+)'", sources['ui']))
    if not ids or not ui or ui - ids:
        raise ValueError('empty or inconsistent demo provider catalogs')
    constructors = {}
    for path in sorted(directory.glob('*.go.tmpl')):
        if path.name.endswith('_test.go.tmpl'):
            continue
        for constructor in re.findall(r'func (New\w+)\(', path.read_text()):
            constructors[constructor] = path
    manager = sources['manager'].split('func (m *Manager) providerForChannel', 1)[1].split('func notificationHeaders', 1)[0]
    direct = {}
    for match in re.finditer(r'case ([^:]+):\n(.*?)(?=\n\tcase |\n\tdefault:)', manager, re.S):
        constructor = re.search(r'return (New\w+)\(', match.group(2))
        if not constructor or constructor.group(1) not in constructors:
            raise ValueError('a dedicated constructor could not be resolved')
        for token in match.group(1).split(','):
            direct[provider(token)] = constructor.group(1)
    if set(direct) - ids:
        raise ValueError('manager provider missing from backend catalog')
    source_paths.update({name: constructors[name] for name in set(direct.values())})
    evidence = []
    for path in sorted(set(source_paths.values())):
        evidence.append({'path': str(path.relative_to(checkout)), 'sha256': hashlib.sha256(path.read_bytes()).hexdigest()})
    commit = subprocess.run(['git', '-C', str(checkout), 'rev-parse', 'HEAD'], check=True, capture_output=True, text=True).stdout.strip()
    entries = []
    catalog = json.loads((ROOT / 'docs/providers/drivers.json').read_text())['drivers']
    library = {entry['id']: entry['package'] + ' (' + entry['verification'] + ')' for entry in catalog}
    library['smtp'] = library['email']
    mail_transports = {'sendgrid': 'mailsendgrid', 'resend': 'mailresend'}
    for name in sorted(ids):
        constructor = direct.get(name)
        status = 'available' if name in library else 'mail_composition_candidate' if name in mail_transports else 'planned'
        entries.append({
            'id': name,
            'demo_implementation': 'dedicated' if constructor else 'generic_webhook_compat',
            'demo_constructor': constructor or 'NewGenericWebhookCompatProvider',
            'demo_adapter': str(constructors[constructor].relative_to(checkout)) if constructor else str(source_paths['generic'].relative_to(checkout)),
            'demo_ui_exposed': name in ui,
            'canonical_id': 'email' if name == 'smtp' else name,
            'library_status': status,
            'library_surface': library.get(name, 'driver/mail with sibling ' + mail_transports[name] + ' (not verified here)' if name in mail_transports else None),
        })
    return {
        'schema_version': 1,
        'source_repository': 'goforj/goforj',
        'source_commit': commit,
        'source_files': evidence,
        'counts': {'backend_ids': len(ids), 'ui_ids': len(ui), 'dedicated_ids': len(direct), 'dedicated_adapters': len(set(direct.values())), 'generic_alias_ids': len(ids - set(direct))},
        'scope': 'Complete demo ID inventory, not a native protocol or hosted-delivery certification.',
        'providers': entries,
    }


def markdown(data):
    """Project the complete inventory without presenting planned providers as supported."""
    counts = data['counts']
    lines = ['# Uptime Gopher provider inventory', '',
             'Generated by `scripts/inventory-demo-providers.py` from the GoForj demo templates.', '',
             f"Source commit: `{data['source_commit']}`. Exact source file hashes are retained in [the JSON snapshot](uptime-gopher.json).", '',
             f"The backend recognizes {counts['backend_ids']} IDs; the UI exposes {counts['ui_ids']}. {counts['dedicated_ids']} IDs use {counts['dedicated_adapters']} dedicated adapters, while {counts['generic_alias_ids']} IDs use the generic webhook compatibility envelope.", '',
             'Dedicated means provider-specific code exists. It does not establish that the current upstream API accepts its payload or that hosted delivery was tested. Generic compatibility means the demo sends its own provider/text/message envelope to a configured URL; it is not native provider support.', '',
             'Library surfaces below distinguish real-server verification from protocol fixtures and pending hosted verification. SendGrid and Resend are candidates for composition through sibling Mail drivers, not newly verified notification integrations.', '',
             '## Dedicated demo adapters', '', '| Provider ID | Demo UI | Current notification library surface |', '| --- | --- | --- |']
    for entry in data['providers']:
        if entry['demo_implementation'] == 'dedicated':
            lines.append(f"| {entry['id']} | {'Yes' if entry['demo_ui_exposed'] else 'No'} | {entry['library_surface'] or 'Planned'} |")
    lines += ['', 'SMTP and Email share the same demo adapter. Their stored provider IDs remain distinct for migration.', '',
              '## Generic webhook compatibility aliases', '', '| Provider ID | Demo UI | Native support status |', '| --- | --- | --- |']
    for entry in data['providers']:
        if entry['demo_implementation'] == 'generic_webhook_compat':
            lines.append(f"| {entry['id']} | {'Yes' if entry['demo_ui_exposed'] else 'No'} | {entry['library_surface'] or 'Requires provider protocol review and implementation'} |")
    lines += ['', f"The extraction scope is all {counts['backend_ids']} IDs in this snapshot, with aliases, retired services, and delegated Mail transports recorded explicitly. See [the provider and API review](../provider-review.md) for sequencing and compatibility decisions.", '']
    return '\n'.join(lines)


def main():
    """Refresh the checked-in snapshot using an explicitly selected framework checkout."""
    parser = argparse.ArgumentParser()
    parser.add_argument('goforj', type=Path, help='GoForj checkout containing templates/demo')
    args = parser.parse_args()
    data = inventory(args.goforj.resolve())
    target = ROOT / 'docs/providers'
    target.mkdir(parents=True, exist_ok=True)
    (target / 'uptime-gopher.json').write_text(json.dumps(data, indent=2) + '\n')
    (target / 'uptime-gopher.md').write_text(markdown(data))
    print(json.dumps(data['counts'], sort_keys=True))


if __name__ == '__main__':
    main()
