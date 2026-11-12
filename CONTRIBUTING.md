# Contributing to xp-provider-gen

Bug reports, fixes, and ideas are welcome. This is a personal project, not a vendored product,
so a reply can take a few days.

## Report a bug

Open an issue with the **Bug report** form. A bug is easiest to act on when it names the
following:

- The output of `xp-provider-gen version`, and how you installed it (release binary,
  `go install`, container image, or a source build).
- The flavor: native, or upjet (`init --upjet`).
- The exact command you ran, with every flag, and the directory state it ran in. For example,
  say whether it was a fresh directory or an existing provider.
- What you expected, and what happened instead, with the full error output.

To report a security problem, do not open an issue. Follow [SECURITY.md](SECURITY.md).

## Suggest a change

For anything larger than a small fix, open an issue first. Describe the problem before the
solution. A change that adds a flag or a template has to earn its place, because every
generated provider carries it.

## Send a pull request

1. Read [AGENTS.md](AGENTS.md). It explains the project's core concepts, the code style, and
   the repository layout. [docs/development.md](docs/development.md) lists the tools you need
   and the usual workflow.
2. Keep the change small and focused on one thing.
3. Run `make reviewable` before you push. It runs the lint, test, security, and consistency
   checks that CI runs, and needs no Docker. If you changed a template, also compare the output of the old and the
   new binary, as AGENTS.md describes. CI also runs the e2e suites on every pull request that is not docs-only. Running them
   locally is optional and needs Docker. [docs/testing.md](docs/testing.md) explains how.
4. Give the pull request a [conventional commit](https://www.conventionalcommits.org/) title,
   such as `docs: fix the upjet quick start in the README`. The
   [`pr-title.yml`](.github/workflows/pr-title.yml) workflow checks the type and the subject.
   It accepts `feat`, `fix`, `perf`, `deps`, `refactor`, `docs`, `test`, `ci`, `chore`, and
   `build`, and a subject in printable ASCII. Mark a breaking change with `!`, as in
   `feat!: remove a flag`, so the release notes list it under breaking changes. If the pull request has a single commit, that
   commit's subject must match the title.

The maintainer squash-merges pull requests, so the title becomes the commit subject on `main`.
The release notes are built from those subjects, so write the title for someone who reads only
the release notes.

## Releases

The maintainer cuts releases from `main` with the Release workflow. Contributors do not tag or
bump versions. The workflow computes the next version and the release notes from the commit
subjects. [.github/WORKFLOWS.md](.github/WORKFLOWS.md) explains the process.
