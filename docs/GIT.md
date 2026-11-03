# Git

Always use the writing skill for commit messages, pull request titles, and pull request descriptions.

## Commits

- Use only the author's configured Git identity and credit. Never add AI co-authors, attribution, trailers, or signatures.
- Write short, direct commit messages in sentence case. Match the style of previous commits.
- Use a single-line commit message. Never add a description or body.

## Pull requests

- Write short, direct PR titles in sentence case, following the same rules as commit messages.
- Keep PR descriptions short, concise, and straightforward.
- Disclose AI involvement only on the final line of the initial PR description, using exactly: `This was written by {model} in {harness} on behalf of the author.` Replace `{model}` with the current model slug and `{harness}` with the harness you are running in. Never use `@` mentions or the author's name or username.
- Strict requirement: never add chat session links (for example `claude.ai/code/session_...`) to PR or issue titles, descriptions, or comments.
- Never add AI attribution anywhere else, including commits, code comments, documentation, or subsequent PR messages.

## Merging

- Before merging, make sure the PR branch is up to date with `main`.
- Always squash all commits into one and use **Squash and merge**. Never use a regular merge commit or rebase-and-merge.
