"""Check that provider inventory snapshots do not promote aliases to native support."""
import importlib.util
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('inventory', Path(__file__).with_name('inventory-demo-providers.py'))
inventory = importlib.util.module_from_spec(spec)
spec.loader.exec_module(inventory)


class InventoryTests(unittest.TestCase):
    """Exercise classification, aliases, provenance and source-drift failures."""

    def fixture(self, root):
        directory = root / 'templates/demo/internal/notification'
        directory.mkdir(parents=True)
        files = {
            'channel_repo.go.tmpl': 'const (\nProviderLog = "log"\nProviderEmail = "email"\nProviderSMTP = "smtp"\n)',
            'provider_catalog.go.tmpl': '''var providers = map[string]struct{}{
    ProviderSMTP: {},
    "matrix": {},
}
func NormalizeProviderID() {}
func IsSupportedNotificationProvider() {
    return normalized == ProviderLog || normalized == ProviderEmail
}
func IsWebhookCompatProvider() {}
''',
            'manager.go.tmpl': '''func (m *Manager) providerForChannel() {
    switch providerID {
\tcase ProviderLog:
        return NewLogProvider()
\tcase ProviderEmail, ProviderSMTP:
        return NewEmailProvider()
\tdefault:
        return nil
    }
}
func notificationHeaders() {}
''',
            'generic_webhook_compat_provider.go.tmpl': 'func NewGenericWebhookCompatProvider() {}',
            'providers.go.tmpl': 'func NewLogProvider() {}',
            'email_provider.go.tmpl': 'func NewEmailProvider() {}',
        }
        for name, content in files.items():
            (directory / name).write_text(content)
        ui = root / 'templates/demo/frontend/src/lib/notification-providers.ts'
        ui.parent.mkdir(parents=True)
        ui.write_text("const webhookCompatProviderIDs: NotificationProvider[] = ['matrix'];\nconst options = [{ id: 'log' }];")
        return directory, ui

    def test_inventory(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            self.fixture(root)
            with patch.object(inventory.subprocess, 'run', return_value=SimpleNamespace(stdout='abc123\n')):
                data = inventory.inventory(root)
            self.assertEqual(data['counts'], {'backend_ids': 4, 'ui_ids': 2, 'dedicated_ids': 3, 'dedicated_adapters': 2, 'generic_alias_ids': 1})
            entries = {entry['id']: entry for entry in data['providers']}
            self.assertEqual(entries['smtp']['canonical_id'], 'email')
            self.assertEqual(entries['matrix']['demo_implementation'], 'generic_webhook_compat')
            self.assertEqual(entries['matrix']['library_status'], 'planned')
            self.assertEqual(data['source_commit'], 'abc123')
            self.assertTrue(all(len(item['sha256']) == 64 for item in data['source_files']))
            rendered = inventory.markdown(data)
            self.assertIn('all 4 IDs', rendered)
            self.assertIn('not native provider support', rendered)

    def test_source_drift(self):
        for defect in ('unknown_ui', 'missing_constructor', 'unknown_manager_id'):
            with self.subTest(defect=defect), tempfile.TemporaryDirectory() as name:
                root = Path(name)
                directory, ui = self.fixture(root)
                if defect == 'unknown_ui':
                    ui.write_text(ui.read_text().replace("'matrix'", "'unknown'"))
                elif defect == 'missing_constructor':
                    (directory / 'email_provider.go.tmpl').write_text('package notification')
                else:
                    path = directory / 'manager.go.tmpl'
                    path.write_text(path.read_text().replace('case ProviderLog:', 'case "unknown":'))
                with self.assertRaises(ValueError):
                    inventory.inventory(root)


if __name__ == '__main__':
    unittest.main()
