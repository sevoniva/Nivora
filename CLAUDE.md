# CLAUDE.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

Nivora uses `AGENTS.md` as the canonical AI coding instruction file.

Read and follow:

- `AGENTS.md`
- `docs/architecture/architecture-contract.md`
- `docs/architecture/module-boundaries.md`
- `docs/engineering/ai-change-policy.md`
- `docs/engineering/dependency-policy.md`
- `docs/engineering/testing-policy.md`
- `docs/engineering/security-baseline.md`
- `docs/engineering/release-scope.md`

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:

- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them; don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If the task is interactive and ambiguity blocks correctness, ask a concise question.
- If the task is autonomous or goal-based, state the assumption, choose the smallest reversible implementation, and document the limitation in the final summary.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it; don't delete it.

When your changes create orphans:

- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:

- "Add validation" -> "Write tests for invalid inputs, then make them pass"
- "Fix the bug" -> "Write a test that reproduces it, then make it pass"
- "Refactor X" -> "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:

```text
1. [Step] -> verify: [check]
2. [Step] -> verify: [check]
3. [Step] -> verify: [check]
```

Strong success criteria let you loop independently. Weak criteria like "make it work" require constant clarification.

## 5. Nivora-Specific Guardrails

- Backend-first. Do not add frontend code unless explicitly requested.
- Current default phase is Phase 0 skeleton unless the user explicitly says otherwise.
- Do not add real cloud, Kubernetes, Argo CD, GitHub, GitLab, Harbor, Nexus, AWS, Aliyun, or Tencent integrations in Phase 0.
- Do not claim production readiness.
- Do not commit secrets or realistic-looking fake credentials.
- Do not violate module boundaries defined in `docs/architecture/module-boundaries.md`.

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.
