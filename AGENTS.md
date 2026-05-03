# AGENTS.md

> Guidance for AI agents working in this repository.

## Project Status

- **Greenfield**: New project, no existing code yet.
- **Name**: flowchat
- **Purpose**: TBD (inferred: flow-based chat application)

## When Code Exists

### Commands
- Check `package.json` scripts, `Makefile`, or task runner config for exact commands.
- Run lint → typecheck → test in that order before committing.

### Architecture
- Read entrypoints first: `src/index.ts`, `src/main.ts`, or framework-specific entry.
- Identify package/module boundaries before making cross-cutting changes.
- Prefer existing patterns over introducing new conventions.

### Style
- Match the project's linting/formatting config (Prettier, ESLint, Biome, etc.).
- Follow TypeScript strict mode if enabled.

### Testing
- Find test config: `vitest.config.ts`, `jest.config.js`, `playwright.config.ts`.
- Run single test: `npx vitest run -t "test name"` or framework equivalent.

### Git
- Small, atomic commits. One logical change per commit.
- No force pushes. No suppressing linter/type errors.
