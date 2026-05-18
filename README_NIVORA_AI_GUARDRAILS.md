# Nivora AI Guardrails Pack

This pack contains project-level guardrails for AI coding agents and human contributors.

Copy or extract these files into the root of the `sevoniva/nivora` repository.

## Included Files

```text
AGENTS.md
CLAUDE.md
.github/copilot-instructions.md
.github/pull_request_template.md
.github/CODEOWNERS
.cursor/rules/nivora.mdc
docs/architecture/architecture-contract.md
docs/architecture/module-boundaries.md
docs/engineering/ai-change-policy.md
docs/engineering/dependency-policy.md
docs/engineering/testing-policy.md
docs/engineering/security-baseline.md
docs/engineering/release-scope.md
scripts/verify-architecture.sh
scripts/verify-no-secrets.sh
.editorconfig
.golangci.yml
```

## Recommended Use

After extracting:

```bash
chmod +x scripts/verify-architecture.sh scripts/verify-no-secrets.sh
./scripts/verify-architecture.sh
./scripts/verify-no-secrets.sh
```

If a file already exists in the repository, compare before overwriting.

## Canonical Rule File

`AGENTS.md` is the canonical AI coding instruction file.

Other tool-specific files should reference or summarize `AGENTS.md`, not create conflicting rules.

## Notes

- `CLAUDE.md` includes the user's original caution-first coding guidelines, with Nivora-specific constraints added.
- `.github/CODEOWNERS` uses placeholder owners. Replace `@sevoniva/maintainers` with actual GitHub users or teams.
- The verification scripts are intentionally conservative and can be expanded once the repository structure stabilizes.
