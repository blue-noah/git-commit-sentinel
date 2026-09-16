# GitHub App: diuis-repo-settings

Runs [safe-settings](https://github.com/github-community-projects/safe-settings) via GitHub Actions to apply
declarative repository settings (rulesets, etc.) from config committed to this repo.

- App ID: `4951095`
- Client ID: `Iv23lidyUXa4TyCjLnD5`
- Installed on: `git-commit-sentinel` (also installed on `chibira-api`, `chibira-doc`, `chibira-web`)

Not stored here (never commit these — GitHub Actions repository secrets only):
- Client secret
- Private key (.pem)

App ID and client ID above are consumed as GitHub Actions **variables** (`vars.SAFE_SETTINGS_APP_ID`,
`vars.SAFE_SETTINGS_GITHUB_CLIENT_ID`), not secrets — they're public identifiers, not credentials.
