# AI Change Policy

This repository is expected to be modified by AI coding agents.

AI-generated changes must follow these rules.

## Before Changing Code

An agent must identify:

- requested goal
- files likely to change
- assumptions
- verification commands

For autonomous tasks, proceed with the smallest reversible assumption.

## During Changes

Do not:

- rewrite unrelated files
- introduce speculative abstractions
- add features outside the requested phase
- add production dependencies without justification
- change public APIs without updating OpenAPI
- alter migrations without understanding schema impact
- add real integrations when placeholders were requested
- add frontend code unless explicitly requested

## After Changes

The agent must report:

- changed areas
- tests run
- build commands run
- failures
- skipped verification with reason
- known limitations

## Safety

Never commit secrets.
Never log secrets.
Never create fake credentials that look real.
Never claim production readiness unless the project explicitly reaches that phase.
