#!/usr/bin/env python3
"""Merge actual unit profiles and executed test events without rounding coverage."""
import json
import subprocess
import importlib.util
from pathlib import Path

root = Path(__file__).resolve().parent.parent
blocks = {}
for path in sorted((root / '.artifacts').glob('unit-[0-9]*.out')) + [root / '.artifacts/examples.out']:
    for line in path.read_text().splitlines()[1:]:
        location, statements, count = line.rsplit(' ', 2)
        statements, count = int(statements), int(count)
        previous = blocks.get(location, (statements, 0))
        if previous[0] != statements:
            raise SystemExit('incompatible coverage instrumentation')
        blocks[location] = statements, previous[1] + count
commands = set()
for module in (root / 'scripts/module-manifest.txt').read_text().splitlines():
    result = subprocess.run(['go', '-C', module, 'list', '-f', '{{.Name}} {{.ImportPath}}', './...'], cwd=root, text=True, capture_output=True, check=True)
    commands.update(line.split()[1] for line in result.stdout.splitlines() if line.startswith('main '))
command_blocks = {key: value for key, value in blocks.items() if key.rsplit('/', 1)[0] in commands}
library_blocks = {key: value for key, value in blocks.items() if key not in command_blocks}
all_profile = 'mode: atomic\n' + ''.join(f'{key} {statements} {count}\n' for key, (statements, count) in sorted(blocks.items()))
(root / '.artifacts/all-unit-coverage.out').write_text(all_profile)
blocks = library_blocks
covered = sum(statements for statements, count in blocks.values() if count > 0)
total = sum(statements for statements, _ in blocks.values())
passed = set()
for path in sorted((root / '.artifacts').glob('unit-*.json')):
    for line in path.read_text().splitlines():
        event = json.loads(line)
        if event.get('Action') == 'pass' and 'Test' in event:
            passed.add((event['Package'], event['Test']))
profile = 'mode: atomic\n' + ''.join(f'{key} {statements} {count}\n' for key, (statements, count) in sorted(blocks.items()))
(root / '.artifacts/unit-coverage.out').write_text(profile)
spec = importlib.util.spec_from_file_location('docs_generator', root / 'scripts/generate-docs.py')
generator = importlib.util.module_from_spec(spec)
spec.loader.exec_module(generator)
evidence = {'source_fingerprint': generator.source_fingerprint(), 'unit_tests_passed': len(passed), 'covered_statements': covered, 'total_statements': total,
            'command_statements': sum(s for s, _ in command_blocks.values()), 'covered_command_statements': sum(s for s, c in command_blocks.values() if c > 0), 'coverage_percent': 100 * covered / total if total else 0, 'modules': (root / 'scripts/module-manifest.txt').read_text().splitlines()}
(root / 'docs/unit-results.json').write_text(json.dumps(evidence, indent=2) + '\n')
print(f'Unit tests passed: {len(passed)}; coverage: {covered}/{total} statements')
if not total or covered != total:
    raise SystemExit('unit statement coverage must be exactly 100%')
