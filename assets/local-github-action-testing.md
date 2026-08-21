# Local GitHub Actions Testing with `act` & Podman

A step-by-step guide to setting up WSL2, Podman, and `act` on Windows to test GitHub Actions workflows locally.

---

## 1. Enable WSL2 Base Feature

Run PowerShell as **Administrator** to enable the core WSL feature needed to power the Podman machine:

```powershell
wsl --install --no-distribution
```

*Reboot your computer if prompted.*

---

## 2. Install Podman Desktop & Initialize Engine

1. Install Podman Desktop using **winget**:
```powershell
winget install RedHat.Podman-Desktop
```


2. Open **Podman Desktop**. On first launch, click **Initialize and Start Podman**. This automatically provisions the dedicated `podman-machine-default` VM and creates the native system pipe integrations.
3. Verify the engine is active in PowerShell:
```powershell
podman system connection list
```



---

## 3. Install `act`

Install `act` via **winget**:

```powershell
winget install nektos.act
```

---

## 4. Select Default Runner Image

Run `act` to generate its initial `.actrc` configuration file:

```powershell
act --list
```

When prompted, choose **Medium** (`catthehacker/ubuntu:act-latest`). This sets the container image used inside the runner to mirror standard GitHub Actions environments (including Go, Node.js, and Git).

---

## 5. Execution Cheatsheet

Run this single-line PowerShell command to clear cached action clones, prevent Windows encoding errors, and execute your workflow:

```powershell
$OutputEncoding = [Console]::OutputEncoding = [System.Text.Encoding]::UTF8; `
Remove-Item -Path "$env:USERPROFILE\.cache\act\beej126*" -Recurse -Force -ErrorAction SilentlyContinue; `
act --bind workflow_dispatch 2>&1 | Where-Object { $_ -notmatch 'unable to get git' }
```

### Useful Options:

* `--bind`: Direct-mounts your repository workspace into the runner container.
* `-j <job_id>`: Runs a specific job from your workflow (e.g., `act -j build`).
* `-s GITHUB_TOKEN="dummy"`: Passes a mock token to suppress step authentication warnings.