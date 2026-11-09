# Development and release

The root module requires Go 1.24 and uses only the standard library. `driver/mail` is an independently publishable optional module using goforj/mail v0.3.1. `examples/mail` is a local example module and is not published independently. `scripts/module-manifest.txt` inventories all four modules.

Run `make test` to execute each module, discover and run canonical example programs, collect child-process coverage, merge actual statement blocks, and enforce exact 100% library unit coverage. Every first-party library package is instrumented, including the optional driver and shared conformance helpers. Example command entry points are executed and output-checked; their startup error handlers are outside the library statement metric. Executed test events, toolchain identity, and statement counts produce `docs/unit-results.json`. Go versions can instrument different statement blocks, so CI enforces exact coverage for each toolchain without requiring their raw statement totals to match the committed report. `make test-race` instruments the child example programs as well as tests. `make test-integration` requires Docker and runs real ntfy and Gotify servers from immutable image digests. It verifies server versions, native acceptance, integer priorities and stored messages. It uses one server per provider per package run and cleans up after failures. The `integration` module isolates Testcontainers dependencies and retains its relative root replacement. Its tagged tests are separate from the unit coverage metric. Successful execution writes `docs/integration-results.json`; hosted services and paired WhatsApp sessions remain pending.

Run `make vet`, `make modules-check`, and `make tidy`. Intentional relative replacements keep optional modules testing unpublished root changes. Tests run with `GOWORK=off`. CI checks that tidy leaves module files unchanged.

README and advanced-guide content comes from `docs/readme.tmpl.md`, `docs/advanced.tmpl.md`, GoDoc symbols, and the marked regions in `examples/*/main.go`. Update these sources, run `make test`, then `make generate` and `make check-generated`. Generation validates API symbols and links to snippets that actually use them. Measured badges come from successful unit evidence, not handwritten counts.

Install the repository-pinned scanner with `go install golang.org/x/vuln/cmd/govulncheck@v1.9.0`, then run `make security` on a patched supported toolchain. CI uses the same scanner pin. Codecov project and patch targets use the unit flag; its GitHub App access and upload token must be configured separately. Set the repository variable `CODECOV_ENABLED=true` only after App access and the `CODECOV_TOKEN` secret are configured.

## Independent release

`make release-plan VERSION=v0.1.0` prints the root and `driver/mail` tag names without creating them. Before an authorized release:

1. Run the complete checks and inspect staged changes.
2. Publish the root tag and verify the module resolves from a clean consumer.
3. Change the optional adapter's root requirement from the development placeholder to that published version. Keep its relative replacement for repository development.
4. Publish the independent `driver/mail/vX.Y.Z` tag and verify it resolves separately.
5. Validate both modules and the Mail example in a temporary external consumer with `GOWORK=off` and no local replacements.

A root tag does not publish the Mail module. No version, release badge, published checksum, hosted provider integration, or Codecov upload is implied by passing local tests.

## Provider expansion research

The [provider and API review](provider-review.md) records the Uptime Gopher inventory, current ecosystem comparison, and proposed extraction sequence. The [complete provider snapshot](providers/uptime-gopher.md) includes all backend IDs without claiming native support for generic webhook aliases.

Refresh the inventory from an explicitly selected framework checkout:

```sh
python3 scripts/inventory-demo-providers.py ../goforj
```

The snapshot records source commit and file hashes. It is a review artifact; normal library CI does not require a sibling framework checkout. Runtime provider support still comes from executed driver contracts and integration evidence.
