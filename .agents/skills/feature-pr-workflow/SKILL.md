---
name: feature-pr-workflow
description: "Use this workflow when implementing a repo change end-to-end: inspect, implement, verify, branch, commit, push, and open a GitHub PR without including secrets or unrelated files."
tools: Read, Edit, Write, Bash, Glob, Grep
model: gpt-5.4
---

You are executing the standard feature delivery workflow for `darts-league`.

Goal:
- take a requested repo change from investigation through implementation, verification, git hygiene, and pull request creation

Use this workflow when:
- the user wants code changes shipped as a branch + commit + PR
- the task spans multiple files or layers
- the user wants tests/builds run before the PR is opened

Do not use this workflow when:
- the user only wants research or a plan
- the user explicitly says not to commit, push, or open a PR
- the task is a trivial one-file change and the user only wants a quick patch

Workflow:

1. Inspect first
- read the relevant code paths before editing
- infer conventions from nearby files, tests, and existing API/UI patterns
- check git status before starting
- if the requirement is ambiguous in a way that changes implementation, ask targeted questions only after doing the non-blocked investigation

2. Plan the slice
- prefer the smallest coherent implementation that fully solves the request
- preserve existing product rules in `AGENTS.md`
- avoid unrelated refactors unless they are required to complete the change safely

3. Implement safely
- make focused changes in the affected backend/frontend layers
- keep files ASCII unless an existing file requires otherwise
- do not introduce secrets, credentials, `.env` contents, tokens, or machine-local config into tracked files
- never stage unrelated changes you did not make

4. Add verification
- add or update tests for the changed behavior when practical
- prefer targeted tests first, then broader suites if the scope warrants it
- run relevant verification commands for the touched areas
- for UI/backend changes, a common default is the backend test suite, frontend unit tests, and frontend build

5. Review before git actions
- inspect `git diff` and `git status`
- confirm only intended files are modified
- check for anything sensitive before staging
- if the worktree contains unrelated user changes, leave them alone and stage only the relevant files

6. Branch and commit
- create a focused branch name based on the change
- stage only relevant files
- write a concise commit message that explains the purpose of the change
- do not amend unless the user explicitly requested it or hooks created follow-up changes on a commit you just made and it is still safe to amend

7. Push and open PR
- push the branch with upstream tracking
- create a PR against `main` unless the repo clearly uses a different base branch
- PR body should include:

```md
## Summary
- <1-3 bullets on the user-visible or architectural change>

## Testing
- <command>
- <command>
```

8. Final response
- report:
  - what changed
  - tests/builds run
  - branch name
  - commit hash
  - PR URL
  - any follow-up risks or logical next steps

Default operating principles:
- ask fewer questions; infer from the repo when safe
- keep the implementation vertical and incremental
- never commit secrets for any reason
- never commit unrelated files for convenience
- leave the repo in a clean, reviewable state

Reusable invocation template:

```text
Implement the following change in this repo:

<describe the change>

Use the feature-pr-workflow:
- inspect current code paths and conventions
- ask questions only if ambiguity materially changes the implementation
- make the smallest clean implementation
- add/update tests
- run relevant tests/builds
- review git diff for scope and secrets
- create a branch, commit only relevant files, push, and open a PR
- return what changed, verification, branch, commit, and PR URL
```
