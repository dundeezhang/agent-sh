# Branch Protection Settings for Main Branch

To configure branch protection for the `main` branch, go to:
**Settings → Branches → Add branch protection rule**

## Recommended Settings:

### Branch name pattern

```text
main
```

### Protection Rules:

✅ **Require a pull request before merging**
  - ✅ Require approvals: **1**
  - ✅ Dismiss stale pull request approvals when new commits are pushed
  - ✅ Require review from Code Owners (@dundeezhang)
  - ✅ Restrict who can dismiss pull request reviews
    - Add: @dundeezhang and repository owners

✅ **Require status checks to pass before merging**
  - ✅ Require branches to be up to date before merging
  - Required status checks:
    - `Test`
    - `Build (ubuntu-latest)`
    - `Build (macos-latest)`
    - `Build (windows-latest)`
    - `Lint`

✅ **Require conversation resolution before merging**

✅ **Require signed commits**

✅ **Require linear history**

✅ **Do not allow bypassing the above settings**

✅ **Restrict who can push to matching branches**
  - Add: @dundeezhang and repository administrators only

❌ **Allow force pushes** — Disabled

❌ **Allow deletions** — Disabled

---

## Quick Setup via GitHub CLI (if available):

```bash
# Install GitHub CLI: https://cli.github.com/

# Enable branch protection
gh api repos/dundeezhang/agent-sh/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["Test","Build (ubuntu-latest)","Build (macos-latest)","Build (windows-latest)","Lint"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":1,"dismiss_stale_reviews":true,"require_code_owner_reviews":true}' \
  --field restrictions='{"users":["dundeezhang"],"teams":[],"apps":[]}' \
  --field required_linear_history=true \
  --field allow_force_pushes=false \
  --field allow_deletions=false \
  --field required_conversation_resolution=true
```

---

## Manual Configuration Steps:

1. Go to: https://github.com/dundeezhang/agent-sh/settings/branches
2. Click "Add branch protection rule"
3. Enter `main` as the branch name pattern
4. Check all the boxes listed above
5. Add @dundeezhang as a required reviewer via CODEOWNERS file (already created)
6. Save changes

**Note:** With these settings, only @dundeezhang or repository owners can approve PRs to merge into main, and all CI checks must pass.
