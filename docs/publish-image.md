# Publish REMI to a container registry (for homelab installers)

Nobody can push to **your** GitHub Container Registry (GHCR) or Docker Hub account without **your** credentials. Use one of the paths below from your machine or CI.

## Path A — GitHub Actions (recommended)

### If `git push` is rejected for the workflow file

GitHub may show: *refusing to allow an OAuth App to create or update workflow … without `workflow` scope* (common when Git Credential Manager or an editor’s sign-in uses a limited OAuth token).

**Fix one of:**

1. **Classic PAT with `workflow` scope** — [Create a token](https://github.com/settings/tokens) with **`repo`** and **`workflow`** (and **`write:packages`** if you also push images manually). Update the remote to use it, or sign in again in Git Credential Manager and choose PAT.
2. **GitHub web UI** — Repository → **Add file** → upload [`.github/workflows/docker-publish.yml`](../.github/workflows/docker-publish.yml) (same contents as in this repo).

After the workflow exists on `main`, routine pushes that do not touch workflow files usually work with your existing credential.

---

1. Push this repository to GitHub (your fork or the canonical repo).
2. Ensure [`.github/workflows/docker-publish.yml`](../.github/workflows/docker-publish.yml) is on the default branch.
3. In the repo: **Settings → Actions → General** — allow **read and write** for workflows (needed for `GITHUB_TOKEN` to push packages).
4. In **Packages** settings for the org/user, allow Actions to publish if prompted.
5. Push to **`main`** → image **`ghcr.io/<lowercase-owner>/remi-trip-planner:latest`** is built and pushed.
6. Tag **`v1.45.0`** (SemVer `v*.*.*`) → additional version tags are pushed alongside `latest`.

**Manual run:** **Actions → Docker Publish → Run workflow**.

### Make the image pullable without login

By default GHCR packages may be **private**. For homelab users who run `docker compose pull` **without** `docker login`:

1. GitHub → **Packages** → **remi-trip-planner** → **Package settings** → **Change visibility** → **Public** (or grant access to collaborators only).

### Homelab install after publish

**No `.env` required** for the official image: use [`docker-compose.install.yml`](../docker-compose.install.yml) or [`docker-compose.registry.yml`](../docker-compose.registry.yml) (defaults to **`ghcr.io/renji61/remi-trip-planner:latest`**).

Optional `.env` for forks or custom tags:

```env
REMI_IMAGE=ghcr.io/your-lowercase-github-username/remi-trip-planner:latest
```

Use your **lowercase** GitHub username or org as shown under the package on GitHub.

---

## Path B — Build and push from your laptop / CI runner

Prerequisites: Docker running, logged in to GHCR.

### 1. Create a GitHub PAT (classic)

Scopes: **`write:packages`**, **`read:packages`**, and often **`delete:packages`** if you replace tags. For user-owned images, **`repo`** may be required depending on org policy.

### 2. Log in to GHCR

**Linux / macOS:**

```bash
echo "$GITHUB_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

**Windows (PowerShell):**

```powershell
$env:GITHUB_TOKEN | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

### 3. Build and push

From the **repository root** (where the `Dockerfile` is):

**Using the helper script**

- **PowerShell:** `.\scripts\publish-ghcr.ps1 -Owner YOUR_GITHUB_USERNAME`
- **Bash:** `./scripts/publish-ghcr.sh YOUR_GITHUB_USERNAME`

Optional: `-Tag v1.0.0` / second argument for a version tag.

**Or manually**

```bash
OWNER=your-github-username   # lowercase
docker build -t ghcr.io/${OWNER}/remi-trip-planner:latest .
docker push ghcr.io/${OWNER}/remi-trip-planner:latest
```

---

## Path C — Other registries (Docker Hub, etc.)

Tag and push to your registry’s naming rules, then set `REMI_IMAGE` in `.env` to that reference. Ensure homelab hosts can reach the registry (`docker login` if private).

---

## Multi-architecture (optional)

The default GitHub Actions workflow builds for the **runner** architecture (usually `linux/amd64`). For **ARM** homelabs (e.g. Raspberry Pi), add **Docker Buildx** + `platforms: linux/amd64,linux/arm64` to the workflow’s `docker/build-push-action` step, or build on an ARM runner.

---

## Verify

```bash
docker pull ghcr.io/renji61/remi-trip-planner:latest
```

Then use [`docker-compose.install.yml`](../docker-compose.install.yml) or [`docker-compose.registry.yml`](../docker-compose.registry.yml) as documented in [self-hosting.md](self-hosting.md).
