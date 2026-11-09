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
        svg = generator.badge('A & B', '100%', icon=True)
        self.assertIn('A &amp; B', svg)
        self.assertIn('<title>', svg)
        self.assertIn('<path', svg)

    def test_projection_and_stale_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'examples/demo').mkdir(parents=True)
            (root / 'examples/demo/main.go').write_text('// snippet:start\nnotification.New()\nfmt.Println("{{a}}")\n// {{a}}\n// snippet:end')
            (root / 'docs').mkdir()
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


if __name__ == '__main__':
    unittest.main()
