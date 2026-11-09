"""Check documentation projections fail safely on stale or incomplete sources."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('generator', Path(__file__).with_name('generate-docs.py'))
generator = importlib.util.module_from_spec(spec)
spec.loader.exec_module(generator)


class GenerationTests(unittest.TestCase):
    """Exercise source ownership and evidence checks without editing the checkout."""

    def test_snippets(self):
        source = '// snippet:start\nfmt.Println("hello")\n// hello\n// snippet:end'
        self.assertIn('// hello', generator.snippet(source))
        for bad in ('', source + source, '// snippet:start\nno output\n// snippet:end'):
            with self.assertRaises(ValueError):
                generator.snippet(bad)

    def test_badge(self):
        svg = generator.badge('A & B', '100%')
        self.assertIn('A &amp; B', svg)
        self.assertIn('<title>', svg)
        self.assertIn('<path', svg)

    def test_projection_and_stale_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'examples/demo').mkdir(parents=True)
            (root / 'examples/demo/main.go').write_text('// snippet:start\nnotification.New()\nfmt.Println("{{a}}")\n// {{a}}\n// snippet:end')
            (root / 'docs/providers').mkdir(parents=True)
            (root / 'docs/assets/driver-icons').mkdir(parents=True)
            (root / 'docs/assets/driver-icons/presentation.json').write_text('{}')
            (root / 'docs/assets/feature-icons').mkdir(parents=True)
            (root / 'docs/assets/feature-icons/presentation.json').write_text('{}')
            (root / 'docs/providers/drivers.json').write_text(json.dumps({'drivers': []}))
            (root / 'docs/readme.tmpl.md').write_text('## Demo\n{{snippet:demo}}\n{{api-index}}')
            with patch.object(generator, 'ROOT', root), patch.object(generator, 'APIS', [('New', '.', 'New', 'demo', 'Demo')]), patch.object(generator.subprocess, 'run'):
                evidence = {'source_fingerprint': generator.source_fingerprint(), 'covered_statements': 1, 'total_statements': 1, 'unit_tests_passed': 1}
                evidence_path = root / 'docs/unit-results.json'
                evidence_path.write_text(json.dumps(evidence))
                self.assertIn('[New](#demo)', generator.outputs()['README.md'])
                (root / 'examples/demo/main.go').write_text('changed')
                with self.assertRaisesRegex(ValueError, 'stale'):
                    generator.outputs()
                evidence['source_fingerprint'] = generator.source_fingerprint()
                evidence['covered_statements'] = 0
                evidence_path.write_text(json.dumps(evidence))
                with self.assertRaisesRegex(ValueError, 'coverage'):
                    generator.outputs()

    def test_driver_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'docs/providers').mkdir(parents=True)
            (root / 'docs/assets/driver-icons').mkdir(parents=True)
            (root / 'docs/assets/driver-icons/presentation.json').write_text('{}')
            (root / 'docs/assets/feature-icons').mkdir(parents=True)
            (root / 'docs/assets/feature-icons/presentation.json').write_text('{}')
            (root / 'integration').mkdir()
            (root / 'docs/readme.tmpl.md').write_text('{{drivers}}\n{{api-index}}')
            entry = {'id': 'ntfy', 'package': 'driver/ntfy', 'configuration': 'topic',
                     'acceptance': 'native acknowledgement', 'verification': 'Real server verified'}
            (root / 'docs/providers/drivers.json').write_text(json.dumps({'drivers': [entry]}))
            (root / 'integration/versions.json').write_text('{}')
            with patch.object(generator, 'ROOT', root), patch.object(generator, 'APIS', []):
                fingerprint = generator.source_fingerprint()
                (root / 'docs/unit-results.json').write_text(json.dumps({
                    'source_fingerprint': fingerprint, 'covered_statements': 1,
                    'total_statements': 1, 'unit_tests_passed': 1}))
                evidence = {'source_fingerprint': fingerprint, 'servers': {}, 'passed_tests': []}
                path = root / 'docs/integration-results.json'
                path.write_text(json.dumps(evidence))
                with self.assertRaisesRegex(ValueError, 'integration evidence is stale'):
                    generator.outputs()
                evidence['passed_tests'] = ['TestNtfy', 'TestGotify']
                path.write_text(json.dumps(evidence))
                with self.assertRaisesRegex(ValueError, 'missing driver package'):
                    generator.outputs()
                (root / 'driver/ntfy').mkdir(parents=True)
                (root / 'docs/assets/driver-icons/presentation.json').write_text(json.dumps({'ntfy': {'label': 'ntfy', 'icon': 'ntfy.svg'}}))
                (root / 'docs/assets/driver-icons/ntfy.svg').write_text('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M0 0h24v24z"/></svg>')
                self.assertIn('> ntfy](driver/ntfy)', generator.outputs()['README.md'])


if __name__ == '__main__':
    unittest.main()
