#!/usr/bin/env python3
"""Run real service contracts and retain evidence only after every required test passes."""
import datetime
import importlib.util
import json
import os
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parent.parent
environment = dict(os.environ, GOWORK='off')
result = subprocess.run(['go', '-C', 'integration', 'test', '-tags=integration', '-count=1', '-json', './...'], cwd=root, env=environment, text=True, capture_output=True)
events = [json.loads(line) for line in result.stdout.splitlines() if line.startswith('{')]
for event in events:
    if event.get('Action') == 'output':
        print(event.get('Output', ''), end='')
if result.returncode:
    print(result.stderr)
    raise SystemExit(result.returncode)
passed = sorted({event['Test'] for event in events if event.get('Action') == 'pass' and 'Test' in event})
if not {'TestNtfy', 'TestGotify'} <= set(passed):
    raise SystemExit('required native integration tests did not execute')
spec = importlib.util.spec_from_file_location('generator', root / 'scripts/generate-docs.py')
generator = importlib.util.module_from_spec(spec)
spec.loader.exec_module(generator)
evidence = {
    'verified_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
    'source_fingerprint': generator.source_fingerprint(),
    'toolchain': subprocess.run(['go', 'version'], text=True, capture_output=True, check=True).stdout.strip(),
    'passed_tests': passed,
    'servers': json.loads((root / 'integration/versions.json').read_text()),
    'scope': 'Real disposable ntfy and Gotify server tests. All other hosted or paired-session verification remains pending.',
}
(root / 'docs/integration-results.json').write_text(json.dumps(evidence, indent=2) + '\n')
