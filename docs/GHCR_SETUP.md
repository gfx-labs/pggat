# GitHub Container Registry (ghcr.io) Setup

## Fixing "permission_denied: write_package" Error

If you're getting this error when GitHub Actions tries to push to ghcr.io:
```
ERROR: failed to push ghcr.io/gfx-labs/pggat:master: denied: permission_denied: write_package
```

Follow these steps to fix it:

## 1. Check Repository Settings

### Enable GitHub Actions Permissions
1. Go to your repository on GitHub: `https://github.com/gfx-labs/pggat`
2. Click **Settings** → **Actions** → **General**
3. Scroll down to **Workflow permissions**
4. Select **"Read and write permissions"**
5. Check **"Allow GitHub Actions to create and approve pull requests"** (optional)
6. Click **Save**

### Enable Package Publishing
1. In repository **Settings** → **Actions** → **General**
2. Under **Workflow permissions**, ensure these are checked:
   - ✅ Read and write permissions
   - ✅ Allow GitHub Actions to write packages

## 2. Check Organization Settings (if applicable)

If your repository is in an organization (`gfx-labs`):

1. Go to organization settings: `https://github.com/organizations/gfx-labs/settings/packages`
2. Check **Package Creation** settings:
   - Ensure **"Members can publish public packages"** is enabled
   - Or add the repository to the allowed list

3. Check **Actions permissions**:
   - Go to `https://github.com/organizations/gfx-labs/settings/actions`
   - Ensure the repository is allowed to use GitHub Actions
   - Check that **"Allow all actions and reusable workflows"** is selected
   - Or add necessary actions to the allow list

## 3. Package Visibility Settings

If the package already exists:

1. Go to `https://github.com/orgs/gfx-labs/packages/container/pggat/settings`
2. Under **Manage Actions access**:
   - Add your repository `gfx-labs/pggat`
   - Grant **Write** permissions

## 4. First-Time Package Creation

If this is the first time pushing the package:

1. The package will be created automatically on first successful push
2. Initially, it will be private (if in an organization with private packages enabled)
3. After creation, you can make it public:
   - Go to the package settings
   - Change visibility to **Public**

## 5. Manual Package Creation (Alternative)

If automatic creation fails:

1. Go to `https://github.com/gfx-labs?tab=packages`
2. Click **"Create a new package"**
3. Choose **Container** type
4. Name it `pggat`
5. Link it to the repository
6. Set visibility (public recommended for open source)

## 6. Verify Workflow Configuration

The workflow already has the correct permissions in `.github/workflows/build.yml`:

```yaml
permissions:
  contents: read
  packages: write  # This is required for pushing to ghcr.io
```

## 7. Alternative: Use Personal Access Token

If repository settings don't work, use a PAT:

1. Create a Personal Access Token:
   - Go to `https://github.com/settings/tokens/new`
   - Select scopes: `write:packages`, `read:packages`, `delete:packages` (optional)
   - Generate token and copy it

2. Add the token as a repository secret:
   - Go to repository **Settings** → **Secrets and variables** → **Actions**
   - Click **"New repository secret"**
   - Name: `GHCR_TOKEN`
   - Value: Your personal access token

3. Update the workflow to use the PAT:
   ```yaml
   - name: Log in to GitHub Container Registry
     uses: docker/login-action@v3
     with:
       registry: ${{ env.REGISTRY }}
       username: ${{ github.actor }}
       password: ${{ secrets.GHCR_TOKEN }}  # Use PAT instead of GITHUB_TOKEN
   ```

## 8. Testing the Fix

After making these changes:

1. Re-run the failed workflow:
   - Go to **Actions** tab
   - Find the failed workflow
   - Click **"Re-run all jobs"**

2. Or trigger a new build:
   ```bash
   git commit --allow-empty -m "Trigger CI"
   git push
   ```

## Common Issues

### Issue: Organization requires SSO
**Solution**: If your organization uses SAML SSO, authorize your PAT for SSO:
1. Go to `https://github.com/settings/tokens`
2. Click **"Configure SSO"** next to your token
3. Authorize for your organization

### Issue: Package name conflicts
**Solution**: The package name must match the pattern. For `ghcr.io/gfx-labs/pggat`:
- Organization: `gfx-labs`
- Package name: `pggat`
- Full image: `ghcr.io/gfx-labs/pggat:tag`

### Issue: Rate limits
**Solution**: GitHub has rate limits for package uploads. If hitting limits:
- Wait for the limit to reset (usually 1 hour)
- Use a PAT with higher rate limits
- Consider using a GitHub App for even higher limits

## Verification

Once working, your packages will be visible at:
- `https://github.com/gfx-labs/pggat/pkgs/container/pggat`
- Or `https://github.com/orgs/gfx-labs/packages/container/package/pggat`

The Docker images can be pulled with:
```bash
docker pull ghcr.io/gfx-labs/pggat:latest
docker pull ghcr.io/gfx-labs/pggat:master
docker pull ghcr.io/gfx-labs/pggat:v1.0.0  # for tagged versions
```