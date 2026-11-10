# Development and release

The root module requires Go 1.24 and uses only the standard library. `driver/mail`, `driver/apns`, `driver/sns` and `driver/webpush` are independently publishable optional modules. Mail uses goforj/mail v0.3.1. Their four matching example modules are local and are not published independently. `docs` is a separate Go tooling module following the sibling library layout, with `gen`, `readme`, `report`, `inventory`, and `modules` packages. It is not a published runtime module. `scripts/module-manifest.txt` inventories all eleven modules.

Run `make test` to execute each module, discover and run canonical example programs, collect child-process coverage, merge actual statement blocks, and enforce exact 100% library unit coverage. Every first-party runtime library package is instrumented, including all optional drivers and shared conformance helpers. The separate documentation tooling module runs its own Go tests and participates in race and vet checks; tooling coverage is separate from the runtime library metric. Example command entry points are executed and output-checked; their startup error handlers are outside the library statement metric. Executed test events, toolchain identity, and statement counts produce `docs/unit-results.json`. Go versions can instrument different statement blocks, so CI enforces exact coverage for each toolchain without requiring their raw statement totals to match the committed report. `make test-race` instruments the child example programs as well as tests. `make test-integration` requires Docker and runs real ntfy, Gotify and Matrix Synapse servers from immutable image digests. It verifies server versions, native acceptance, integer priorities, authenticated Matrix room-version-12 messages and stored messages. It uses one server per provider per package run and cleans up after failures. The `integration` module isolates Testcontainers dependencies and retains its relative root replacement. Its tagged tests are separate from the unit coverage metric. Successful execution writes `docs/integration-results.json`; hosted services and paired Signal sessions remain pending.

Run `make vet`, `make modules-check`, and `make tidy`. Runtime requirements now pin public, proxy-resolvable revisions. Intentional relative replacements keep optional modules testing unpublished root changes. Tests run with `GOWORK=off`. CI checks that tidy leaves module files unchanged.

`docs/release.json` owns the version shown by the release badge and the tagged quick-start checkout. README installation commands use `@latest`; release notes name explicit versions for reproducibility. Update the release metadata before preparing a release, then regenerate the documentation.

`make generate` runs `go -C docs run ./readme`. `make check-generated` checks freshness and two deterministic generations, including unchanged example programs. README, driver-guide and advanced-guide content comes from `docs/readme.tmpl.md`, `docs/advanced.tmpl.md`, GoDoc symbols, and the marked regions in `examples/*/main.go`. Update these sources, run `make test`, then `make generate` and `make check-generated`. Generation reads Go declarations with the standard Go AST packages and validates API links against calls in the linked snippets. `docs/report` merges Go coverage and executed test events; source fingerprints captured before each run prevent old artifacts from being relabelled as new evidence. The driver catalog supplies concise acceptance summaries for the README and the full configuration, acknowledgement and verification details in `docs/drivers.md`. Feature labels, descriptions and artwork come from `docs/assets/feature-icons/presentation.json`. Measured badges come from successful unit evidence, not handwritten counts. The constructor index follows the complete driver catalog. Its configuration expressions are extracted with Go AST parsing from the same runnable programs, including their tested provider configuration helpers. Generation fails when a catalogued driver has no matching constructor call, exported symbol or README example destination.

Install the repository-pinned scanner with `go install golang.org/x/vuln/cmd/govulncheck@v1.9.0`, then run `make security` on a patched supported toolchain. CI uses the same scanner pin. Codecov project and patch targets use the unit flag; its GitHub App access and upload token must be configured separately. Set the repository variable `CODECOV_ENABLED=true` only after App access and the `CODECOV_TOKEN` secret are configured.

## Independent release

`make release-plan VERSION=v0.1.0` prints the root and all four optional module tag names without creating them. Before an authorized release:

1. Run the complete checks, inspect staged changes, and run `make verify-consumer REVISION=<pushed-commit>`.
2. Publish the root tag and verify the module resolves from a clean consumer.
3. Change each optional module's root requirement from its pinned public revision to that published version. Update the example and integration requirements as applicable. Keep its relative replacement for repository development.
4. Publish independent `driver/mail/vX.Y.Z`, `driver/apns/vX.Y.Z`, `driver/sns/vX.Y.Z` and `driver/webpush/vX.Y.Z` tags and verify each resolves separately.
5. Run `make verify-consumer REVISION=vX.Y.Z` to resolve all five runtime tags through the public Go proxy and checksum database, compile every catalogued driver, and exercise Sender with a fresh module cache, GOWORK=off and no replacements. Keep the example modules' version pins aligned.

Do not change optional requirements to an unpublished root release before its tag is available: the public consumer check would correctly fail. Commit and verify those requirement changes after publishing the root, before tagging the optional modules.

The public consumer check is implemented in Go under `docs/consumer` and `docs/gen/consumer.go`. CI runs it for each push after the quality and native integration jobs pass. It resolves the actual pushed commit without rewriting requirements, disables private-module bypasses and local overlays, and uses the standard public checksum database. It checks a combined consumer plus the root and every optional module independently, so an explicitly upgraded root cannot hide an adapter's invalid minimum requirement. The consumers also exercise invalid-destination rejection and HTTP 408/500 uncertainty with a local transport, so compiling alone cannot hide an outdated behavioral dependency. A successful candidate-archive test alone does not establish public installation.

A root tag does not publish the optional modules. No version, release badge, published checksum, hosted provider integration, or Codecov upload is implied by passing local tests.

## Provider expansion research

The [provider and API review](provider-review.md) records the Uptime Gopher inventory, current ecosystem comparison, and proposed extraction sequence. The [complete provider snapshot](providers/uptime-gopher.md) includes all backend IDs without claiming native support for generic webhook aliases.

Refresh the inventory from an explicitly selected framework checkout:

```sh
go -C docs run ./inventory -goforj ../goforj
```

The snapshot records source commit and file hashes. It is a review artifact; normal library CI does not require a sibling framework checkout. Runtime provider support still comes from executed driver contracts and integration evidence.
