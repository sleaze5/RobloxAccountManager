# Git

Always use the writing skill for commit messages, pull request titles, and pull request descriptions.

## Commits

- Use only the author's configured Git identity and credit. Never add AI co-authors, attribution, trailers, or signatures.
- Write short, direct commit messages in sentence case. Match the style of previous commits.
- Use a single-line commit message. Never add a description or body.

## Pull requests & issues

- Write short, direct titles in sentence case, following the same rules as commit messages.
- Keep the descriptions short, concise, and straightforward.
- Disclose AI involvement on the last line of the initial description, using exactly the line below. Replace `{model_slug}` with the current model slug and `{harness}` with the harness it is running in.
    ```markdown
    This was written by `{model_slug}` in `{harness}` on behalf of the author.
    ```
- When you edit a pull request or issue description while reviewing it, add one line at the end of the description, using exactly the line below. Keep the author's text unchanged.
    ```markdown
    This was also edited by `{model_slug}` in `{harness}`.
    ```
- Never add chat session links (ie. `claude.ai/code/session_...`, `chatgpt.com/c/...`) to titles, descriptions, or comments.
- Never add AI attribution anywhere else, including commits, code comments, documentation, or subsequent messages, except the edit line above.

## Merging

- Before merging, make sure the PR branch is up to date with `main`.
- Always squash all commits into one and use **Squash and merge**. Never use a regular merge commit or rebase-and-merge.
