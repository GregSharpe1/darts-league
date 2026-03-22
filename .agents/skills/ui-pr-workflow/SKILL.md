---
name: ui-pr-workflow
description: "Use this workflow for user-visible frontend changes: run full frontend verification, capture tracked screenshots, and open a PR with testing and screenshot evidence."
tools: Read, Write, Bash, Glob, Grep
model: gpt-5.4
---

You are executing the UI delivery workflow for `darts-league`.

Goal:
- take a user-visible UI change from investigation through implementation, verification, screenshot capture, git hygiene, and pull request creation

Use this workflow when:
- the change affects visible frontend behavior under `frontend/`
- the task changes layout, styling, copy, navigation, interaction flows, or admin/public page states
- the user wants a UI change shipped as a branch + commit + PR with evidence

Do not use this workflow when:
- the user only wants research or a plan
- the change is backend-only or invisible refactoring with no user-visible effect
- the user explicitly says not to commit, push, or open a PR

Workflow:

1. Inspect first
- read the affected UI routes, nearby tests, and any dependent API calls before editing
- check `git status` before starting
- infer conventions from the existing frontend and from `.agents/skills/feature-pr-workflow/SKILL.md`
- if a requirement is ambiguous in a way that changes implementation, ask only after non-blocked investigation

2. Plan the slice
- prefer the smallest coherent UI change that fully solves the request
- preserve the product rules in `AGENTS.md`
- avoid unrelated refactors unless they are required to complete the UI work safely

3. Implement safely
- make focused frontend changes and any minimal backend/test updates required to support them
- keep files ASCII unless an existing file requires otherwise
- never introduce secrets, credentials, `.env` contents, tokens, or machine-local config into tracked files
- never stage unrelated changes you did not make

4. Keep UI verification current
- add or update automated coverage for the changed user flow when practical
- UI verification must include a Playwright flow that covers at least:
  - visiting `/register`
  - entering at least 8 players
  - visiting `/admin`
  - logging in as admin
  - starting a new season
  - verifying at least one post-start public state, preferably `/`, `/standings`, or the closed `/register` state
- if the UI change touches registration, admin auth, season start, standings, fixtures, or result entry, run backend tests too

5. Run required verification
- always run the full frontend verification suite:
  - `npm test`
  - `npm run build`
  - `npm run test:e2e:ui`
- run `go test ./...` from `backend/` when the UI work depends on backend state or API behavior
- do not skip failed checks; fix the issue or clearly report the blocker

6. Capture tracked screenshots
- UI PRs must include fresh screenshots committed in `docs/pr-screenshots/<branch-name>/`
- use the current git branch name, sanitized for file paths if needed
- capture at minimum:
  - `register-open.png`
  - `admin-pre-start.png`
  - `admin-post-start.png`
  - `public-post-start.png`
- add extra feature-specific screenshots when the changed UI area is not already covered by the required set
- confirm the screenshot files are updated before staging
- after pushing the branch, embed the screenshots directly in the PR body, not just as path bullets
- build image URLs from the pushed repo slug plus the pushed HEAD commit SHA so the rendered images are stable, for example: `https://raw.githubusercontent.com/<owner>/<repo>/<commit-sha>/docs/pr-screenshots/<branch-name>/register-open.png`
- keep the file-path bullets in the PR body too, but add rendered markdown images underneath them so reviewers can see the screenshots inline on GitHub

7. Review before git actions
- inspect `git diff` and `git status`
- confirm only intended code, test, and screenshot files are modified
- check for anything sensitive before staging
- if the worktree contains unrelated user changes, leave them alone and stage only the relevant files

8. Branch, commit, push, and open PR
- create a focused branch name if needed
- stage only relevant files, including the tracked screenshots
- write a concise commit message that explains the purpose of the UI change
- push with upstream tracking
- create a PR against `main` unless the repo clearly uses a different base branch
- PR body should include:

```md
## Summary
- <1-3 bullets on the user-visible UI change>

## Testing
- npm test
- npm run build
- npm run test:e2e:ui
- <optional backend test command>

## Screenshots
![Register open](https://raw.githubusercontent.com/<owner>/<repo>/<branch-name>/docs/pr-screenshots/<branch-name>/register-open.png)
![Admin pre start](https://raw.githubusercontent.com/<owner>/<repo>/<branch-name>/docs/pr-screenshots/<branch-name>/admin-pre-start.png)
![Admin post start](https://raw.githubusercontent.com/<owner>/<repo>/<branch-name>/docs/pr-screenshots/<branch-name>/admin-post-start.png)
![Public post start](https://raw.githubusercontent.com/<owner>/<repo>/<branch-name>/docs/pr-screenshots/<branch-name>/public-post-start.png)
```

- prefer embedded markdown images in the PR body so reviewers can see the screenshots without clicking through
- docs/pr-screenshots/<branch-name>/register-open.png
- docs/pr-screenshots/<branch-name>/admin-pre-start.png
- docs/pr-screenshots/<branch-name>/admin-post-start.png
- docs/pr-screenshots/<branch-name>/public-post-start.png

### Register Open
![Register open](https://raw.githubusercontent.com/<owner>/<repo>/<commit-sha>/docs/pr-screenshots/<branch-name>/register-open.png)

### Admin Pre Start
![Admin pre start](https://raw.githubusercontent.com/<owner>/<repo>/<commit-sha>/docs/pr-screenshots/<branch-name>/admin-pre-start.png)

### Admin Post Start
![Admin post start](https://raw.githubusercontent.com/<owner>/<repo>/<commit-sha>/docs/pr-screenshots/<branch-name>/admin-post-start.png)

### Public Post Start
![Public post start](https://raw.githubusercontent.com/<owner>/<repo>/<commit-sha>/docs/pr-screenshots/<branch-name>/public-post-start.png)
```

- before calling `gh pr create`, gather the repo owner/name with `gh repo view --json owner,name` and the pushed commit SHA with `git rev-parse HEAD`, then interpolate those values into the markdown image URLs above

9. Final response
- report:
  - what changed
  - tests/builds run
  - screenshot directory
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
Implement the following UI change in this repo:

<describe the change>

Use the ui-pr-workflow:
- inspect current UI code paths and conventions
- ask questions only if ambiguity materially changes implementation
- make the smallest clean implementation
- add or update UI coverage
- run npm test, npm run build, and npm run test:e2e:ui
- run backend tests too when the UI depends on backend behavior
- capture and commit screenshots in docs/pr-screenshots/<branch-name>/
- review git diff for scope and secrets
- create a branch, commit only relevant files, push, and open a PR
- return what changed, verification, screenshot directory, branch, commit, and PR URL
```
