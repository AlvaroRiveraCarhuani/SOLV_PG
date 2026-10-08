# Docker Socket Access & Container Isolation Security Audit

## Overview

The SOLV platform orchestrates dynamic virtual lab containers and code evaluation runners using Docker Engine. Because access to the Docker daemon socket (`/var/run/docker.sock`) grants administrative control over the host system, this audit establishes the permissions model, container isolation safeguards, and access constraints.

---

## 1. Docker Socket Access Model

### Host Permissions
- **Socket Path:** `/var/run/docker.sock`
- **Required File Permissions:** `srw-rw----`
- **Ownership:** `root:docker`
- **Backend Service Access:** The SOLV backend process runs under a dedicated system user belonging to the `docker` group.

### Security Controls & Mitigation
1. **No Docker Socket Mounting in Runner Containers:**
   - Ephemeral student evaluation containers and OpenVSCode workspace containers are strictly **forbidden** from having `/var/run/docker.sock` mounted into their filesystems.
   - Container breakout attempts inside student/workspace containers cannot access the Docker daemon API.

2. **Backend API Scope Isolation:**
   - Only the trusted backend API daemon interacts with the Docker SDK (`github.com/docker/docker/client`).
   - Container creation parameters enforce non-root execution, cgroups constraints, read-only root filesystems where applicable, and memory/CPU limits.

---

## 2. Ephemeral Evaluation Container Hardening

Evaluation containers executed by the judge service enforce the following security constraints:

- **Security Profile (seccomp):** Default Docker seccomp profile blocking sensitive system calls (`ptrace`, `sys_admin`, etc.).
- **Capabilities Drop:** All Linux capabilities dropped (`--cap-drop=ALL`).
- **Resource Constraints (cgroups):**
  - Memory limit: 256 MB hard ceiling (OOM kill triggered if exceeded).
  - CPU quota: 1.0 CPU max quota.
  - Process limit (pids-limit): Maximum 64 processes/threads (prevents fork bombs).
- **Network Isolation:** Ephemeral evaluation containers run with network disabled (`--net=none`), preventing unauthorized outbound network calls or data exfiltration.

---

## 3. Workspace Container Isolation

OpenVSCode Server workspace containers are isolated per student session:

- **Port Binding:** Ports are bound strictly to `127.0.0.1` (localhost) and reverse-proxied through Traefik with authentication verification.
- **Storage Volumes:** Ephemeral volume mounts restricted to the user's specific workspace directory (`/home/coder/workspace`). Host filesystem mounts outside workspace boundaries are prohibited.
- **Lifetime & Cleanup:** Inactive workspace containers are automatically paused or terminated by the QoS worker and zombie collector workers.
