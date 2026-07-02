# MicroShift Development Environment

Build and test MicroShift (RPMs, bootc images, ISOs, VMs) inside a libvirt VM
with NFS-shared sources from any Linux host with KVM.

## Prerequisites

- RHEL or Fedora host with KVM (`/dev/kvm`)
- `podman`, `git`, `jq` installed
- Red Hat pull secret at `~/.pull-secret.json` (or set `PULL_SECRET`)
- RHSM credentials (`RHSM_ORG` and `RHSM_ACTIVATION_KEY` environment variables)

## Quick Start

```bash
export RHSM_ORG="your-org-id"
export RHSM_ACTIVATION_KEY="your-activation-key"

# Build the VM base image (bootc container → qcow2)
./scripts/devenv-builder/devenv.sh setup

# Create and start the VM
./scripts/devenv-builder/devenv.sh start

# Open a shell inside the VM
./scripts/devenv-builder/devenv.sh shell

# Build RPMs (inside the VM)
make rpm
```

## Commands

| Command | Description |
|---------|-------------|
| `setup` | Build bootc container image, convert to qcow2 base disk |
| `start` | Create VM from base disk, mount NFS, register subscription |
| `stop` | Graceful shutdown (force after 60s timeout) |
| `delete` | Remove the VM definition (base disk image preserved) |
| `shell` | SSH into the VM as the `microshift` user |
| `exec` | Run a command in the VM as the `microshift` user |
| `status` | Show VM state and IP address |

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `RHSM_ORG` | setup, start | -- | Red Hat subscription org ID |
| `RHSM_ACTIVATION_KEY` | setup, start | -- | Red Hat subscription activation key |
| `PULL_SECRET` | setup | `~/.pull-secret.json` | Path to OpenShift pull secret |
| `DEVENV_BRANCH` | -- | current branch | Target branch for the build |
| `VM_CPUS` | -- | `4` | Number of vCPUs |
| `VM_MEMORY` | -- | `8192` | Memory in MiB |

## Per-Release Builds

Each branch maps to a RHEL version in `rhel-versions.json`. Use `DEVENV_BRANCH`
to build for a different release:

```bash
DEVENV_BRANCH=release-4.21 ./scripts/devenv-builder/devenv.sh setup
DEVENV_BRANCH=release-4.21 ./scripts/devenv-builder/devenv.sh start
DEVENV_BRANCH=release-4.21 ./scripts/devenv-builder/devenv.sh shell
```

Each branch gets its own VM (`microshift-devenv-release-4.21`) and base image,
so multiple releases can run side by side.

## How It Works

### Setup (image build)

1. Creates a git worktree at `.worktrees/microshift-devenv-<branch>` so that
   `configure-vm.sh` and `configure-composer.sh` come from the target branch.
2. Builds a RHEL bootc container image (`Containerfile.vm`) that:
   - Creates a `builder` user with SSH key and console password
   - Registers RHSM, runs `configure-vm.sh --no-build` and
     `configure-composer.sh` to install all build dependencies, then
     unregisters so credentials are not baked into the image
   - Redirects `/tmp` to `/var/tmp` (composefs root is read-only)
3. Converts the bootc image to a qcow2 disk using `image-builder-cli` with a
   blueprint that disables zram (`systemd.zram=0`).
4. Stores the base image at `_output/microshift-devenv-<branch>/base.qcow2`.

### Start (VM creation)

1. Copies the base image to a working disk and resizes it to 50 GiB
   (`bootc-generic-growpart` expands `/var` on first boot).
2. Exports the project root via NFS and opens firewall ports
   (nfs, mountd, rpc-bind) on the libvirt zone.
3. Creates the VM with `virt-install` on the default libvirt network.
4. Mounts the NFS export at `/var/microshift` inside the VM.
5. Creates a `microshift` user matching the host UID/GID for file ownership
   consistency across the NFS mount.
6. Copies the pull secret and registers the RHSM subscription.

### Stop / Delete

- `stop` attempts a graceful `virsh shutdown`, falls back to `virsh destroy`
  after 60 seconds.
- `delete` removes the VM definition but preserves the base disk image. Use
  `delete` + `start` for a clean VM from the same base.

## Editing Code

The NFS mount provides bidirectional, real-time file sharing. Edit on the host
in your IDE, build inside the VM:

```bash
# On the host — edit as usual
vim pkg/config/config.go

# In the VM shell — build immediately sees the change
./scripts/devenv-builder/devenv.sh exec make rpm
```

## Debugging

- **Console access:** `sudo virsh console microshift-devenv-<branch>` with the
  password stored in `_output/microshift-devenv-<branch>/builder_password`.
- **virt-manager:** The VM is a standard libvirt domain, visible in
  `virt-manager` for graphical console and resource monitoring.
- **SSH key:** Stored at `_output/microshift-devenv-<branch>/ssh_key`.
