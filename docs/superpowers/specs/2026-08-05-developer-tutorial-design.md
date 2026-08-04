# Developer Tutorial — Design

- **Date:** 2026-08-05
- **Status:** Approved (design); ready for implementation planning
- **Scope:** One new hands-on document, `docs/tutorial.md`, plus cross-links. No CLI,
  template, or test-harness changes.
- **Builds on:** [2026-07-31 Modular Provider Layout](2026-07-31-modular-provider-layout-design.md)
  (the seam contract the tutorial teaches) and `docs/provider-guide.md` (the reference
  the tutorial links into).

## 1. Problem

The docs explain the modular layout as *reference*: README gives commands,
`provider-guide.md` says where code goes. Nothing demonstrates the journey — a
developer cannot yet follow one document from an empty directory to a provider that
reconciles a real resource, then take a generator upgrade. "How do I actually use
this?" has no answer that ends with something running.

## 2. Decision

Add `docs/tutorial.md`: a followable walkthrough (~30 min) building **provider-local**,
a provider whose external system is the local kind cluster itself — each managed
`Bucket` (group `storage`) materializes as a **ConfigMap** managed through the
`cfg.Kube` client the connector already injects.

Why ConfigMap-as-external-resource:

- **Fully local.** No cloud account, no mock API server to run. The user requirement:
  the example must work locally, verified on a kind cluster.
- **Observable.** Every CRUD step verifies with `kubectl get configmap`.
- **Teaches the real seams.** It exercises every user-owned file for real: kind types,
  a user-added `ProviderConfigSpec` field, `NewClient` from `ClientConfig`, and all
  four `External` methods. Credentials use `source: None`, demonstrating the
  ambient-identity path.

Rejected: a tiny HTTP server shipped with the tutorial (extra moving part to run and
explain); a filesystem-backed example (nothing observable in the cluster).

## 3. The tutorial's content

Sections, in reading order:

1. **What you'll build & prerequisites** — Go, kind, kubectl, Docker (for kind);
   the ownership model in three sentences with a link to `provider-guide.md`.
2. **Scaffold** — build/install the generator, then
   `init --domain=example.com --repo=github.com/example/provider-local` and
   `create api --group=storage --version=v1alpha1 --kind=Bucket`.
3. **Define your API** — edit `apis/storage/v1alpha1/bucket_types.go`:
   `spec.forProvider.data map[string]string`, observed state in `status.atProvider`;
   run `make generate`.
4. **Extend the ProviderConfig** — add `Namespace string` to `ProviderConfigSpec` in
   `apis/v1alpha1/types.go` (where the provider writes ConfigMaps); `make generate`.
5. **Build your client** — `internal/provider/client.go`: `Client` wraps `cfg.Kube`
   plus the namespace from `cfg.Spec.Namespace` (defaulting to `default`).
6. **Implement the external client** — `internal/controller/bucket/external.go`:
   Observe (get ConfigMap by external-name; up-to-date = data matches), Create,
   Update, Delete. External-name is the ConfigMap name.
7. **Unit test it** — one table-driven test using controller-runtime's fake client
   (`sigs.k8s.io/controller-runtime/pkg/client/fake`, already in the module graph);
   `go test ./...`.
8. **Run it on kind** — `make dev` (creates the kind cluster, installs CRDs, runs the
   controller locally), apply a ProviderConfig (`credentials.source: None`, the
   user-added `namespace` field) and a Bucket manifest, watch the ConfigMap appear;
   change spec data → ConfigMap updates; delete the Bucket → ConfigMap goes;
   `make dev-clean`.
9. **Take an upgrade** — `xp-provider-gen update` in brief; link to the guide's
   Upgrading section rather than repeating it.
10. **Where to go next** — `docs/ownership.md` and `AGENTS.md` in the generated
    provider; `provider-guide.md` for the full seam reference.

Style rules:

- **Every code block is code that was actually run.** The verification (§5) executes
  the tutorial; the doc's snippets are copied from the working result, never written
  free-hand.
- **DRY with the guide.** Concepts the guide already explains (seam names, update
  contract, flags) are linked, not restated.

## 4. Cross-links

- `README.md` Quick Start gains one line pointing to the tutorial.
- `docs/provider-guide.md` intro gains one line: "Prefer a walkthrough? See
  [the tutorial](tutorial.md)."
- `CLAUDE.md`'s deeper-docs list gains the tutorial entry.

## 5. Verification (user requirement: verify locally, then run the e2e)

1. **Execute the tutorial end-to-end on this machine** — scaffold, apply each code
   edit exactly as the doc shows, `make reviewable` in the generated provider,
   `go test ./...`, then the kind section: `make dev`, apply ProviderConfig + Bucket,
   assert the ConfigMap exists with the right data, update spec → assert data
   changed, delete → assert gone, `make dev-clean`.
2. **Run the repo harnesses afterward** — `make e2e-test` (and the local gate
   `fmt vet lint test`) must stay green. The tutorial changes only docs, so this
   proves no accidental repo damage.

## 6. Non-goals

- No CI job that executes the tutorial (manual verification this once; the layout
  facts it depends on are guarded by `e2e-test`/`upgrade-sim`).
- No changes to templates, CLI, or test harnesses.
- No CLI-flag implementation in the tutorial's main path (guide link instead).
