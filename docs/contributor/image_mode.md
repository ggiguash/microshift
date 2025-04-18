# Image Mode for MicroShift

Image mode is a new approach to operating system deployment that lets users build,
deploy, and manage Red Hat Enterprise Linux as a bootable container (`bootc`)
image. Such an image uses standard OCI/Docker containers as a transport and delivery
format for base operating system updates. A `bootc` image includes a Linux kernel,
which is used to boot.

MicroShift build and deployment procedures can utilize bootable containers to
benefit from this technology.

This document demonstrates how to:
* Create a bootable container image and boot RHEL from this image using `podman`
* Store a `bootc` image in a remote registry and use it for installing a new RHEL
operating system

> See [Image mode for Red Hat Enterprise Linux](https://developers.redhat.com/products/rhel-image-mode/overview)
for more information.

The procedures described below require the following setup:
* A `RHEL 9.4 host` with an active Red Hat subscription for building MicroShift `bootc`
images and running containers
* A `physical hypervisor host` with the [libvirt](https://libvirt.org/) virtualization
platform for starting virtual machines that run RHEL OS containing MicroShift binaries
* A `remote registry` (e.g. `quay.io`) for storing and accessing `bootc` images

## Build MicroShift Bootc Image

Log into the `RHEL 9.4 host` using the user credentials that have SUDO
permissions configured.

### Build Image

Download the [Containerfile](../config/Containerfile.bootc-rhel9) using the following
command and use it for subsequent image builds.

```bash
URL=https://raw.githubusercontent.com/openshift/microshift/refs/heads/main/docs/config/Containerfile.bootc-rhel9

curl -s -o Containerfile "${URL}"
```

> **Important:**<br>
> When building a container image, podman uses host subscription information and
> repositories inside the container. With only `BaseOS` and `AppStream` system
> repositories enabled by default, we need to enable the `rhocp` and `fast-datapath`
> repositories for installing MicroShift and its dependencies. These repositories
> must be accessible in the host subscription, but not necessarily enabled on the host.

Run the following image build command to create a local `bootc` image.

Note how secrets are used during the image build:
* The podman `--authfile` argument is required to pull the base image from the
`registry.redhat.io` registry
* The build `USER_PASSWD` argument is used to set a password for the `redhat` user

```bash
PULL_SECRET=~/.pull-secret.json
USER_PASSWD="<your_redhat_user_password>"
IMAGE_NAME=microshift-4.18-bootc

sudo podman build --authfile "${PULL_SECRET}" -t "${IMAGE_NAME}" \
    --build-arg USER_PASSWD="${USER_PASSWD}" \
    -f Containerfile
```

> **Important:**<br>
> If `dnf upgrade` command is used in the container image build procedure, it
> may cause unintended operating system version upgrade (e.g. from `9.4` to
> `9.6`). To prevent this from happening, use the following command instead.
> ```
> RUN . /etc/os-release && dnf upgrade -y --releasever="${VERSION_ID}"
> ```

Verify that the local MicroShift 4.18 `bootc` image was created.

```bash
$ sudo podman images "${IMAGE_NAME}"
REPOSITORY                       TAG         IMAGE ID      CREATED        SIZE
localhost/microshift-4.18-bootc  latest      193425283c00  2 minutes ago  2.31 GB
```

### Publish Image

Run the following commands to log into a remote registry and publish the image.

> The image from the remote registry can be used for running the container on
> another host, or when installing a new operating system with the `bootc`
> image layer.

```bash
REGISTRY_URL=quay.io
REGISTRY_IMG=myorg/mypath/"${IMAGE_NAME}"

sudo podman login "${REGISTRY_URL}"
sudo podman push localhost/"${IMAGE_NAME}" "${REGISTRY_URL}/${REGISTRY_IMG}"
```

> Replace `myorg/mypath` with your remote registry organization name and path.

## Run MicroShift Bootc Image

Log into the `RHEL 9.4 host` using the user credentials that have SUDO
permissions configured.

### Configure CNI

The MicroShift CNI driver (OVN) requires the Open vSwitch service to function
properly. The service depends on the `openvswitch` kernel module to be available
in the `bootc` image.

Run the following commands on the host to check the `openvswitch` module presence
and the version of `kernel-core` package used in the `bootc` image. Note that the
kernel versions are different.

```bash
$ find /lib/modules/$(uname -r) -name "openvswitch*"
/lib/modules/6.9.9-200.fc40.x86_64/kernel/net/openvswitch
/lib/modules/6.9.9-200.fc40.x86_64/kernel/net/openvswitch/openvswitch.ko.xz

$ IMAGE_NAME=microshift-4.18-bootc
$ sudo podman inspect "${IMAGE_NAME}" | grep kernel-core
        "created_by": "kernel-core-5.14.0-427.26.1.el9_4.x86_64"
```

When a `bootc` image is started as a container, it uses the host kernel, which is
not necessarily the same one used for building the image. This means that the
`openvswitch` module cannot be loaded in the container due to the kernel version
mismatch with the modules present in the `/lib/modules` directory.

One way to work around this problem is to pre-load the `openvswitch` module before
starting the container as described in the [Run Container](#run-container) section.

### Configure CSI

If the host is already configured to have a `rhel` volume group with free space,
this configuration is inherited by the container so that it can be used by the
MicroShift CSI driver to allocate storage.

Run the following command to determine if the volume group exists and it has the
necessary free space.

```bash
$ sudo vgs
  VG   #PV #LV #SN Attr   VSize   VFree
  rhel   1   1   0 wz--n- <91.02g <2.02g
```

Otherwise, a new volume group should be set up for MicroShift CSI driver to allocate
storage in `bootc` MicroShift containers.

Run the following commands to create a file to be used for LVM partitioning and
configure it as a loop device.

```bash
VGFILE=/var/lib/microshift-lvm-storage.img
VGSIZE=1G

sudo truncate --size="${VGSIZE}" "${VGFILE}"
sudo losetup -f "${VGFILE}"
```

Query the loop device name and create a free volume group on the device according
to the MicroShift CSI driver requirements described in [Storage Configuration](./storage/configuration.md).

```bash
VGLOOP=$(losetup -j ${VGFILE} | cut -d: -f1)
sudo vgcreate -f -y rhel "${VGLOOP}"
```

The device will now be shared with the newly created containers as described in
the next section.

> The following commands can be run to detach the loop device and delete the LVM
> volume group file.
>
> ```bash
> sudo losetup -d "${VGLOOP}"
> sudo rm -f "${VGFILE}"
> ```

### Run Container

Run the following commands to start the MicroShift `bootc` image in an interactive
terminal session.

The host shares the following configuration with the container:
* The `openvswitch` kernel module to be used by the Open vSwitch service
* A pull secret file for downloading the required OpenShift container images
* Host container storage for reusing available container images

```bash
PULL_SECRET=~/.pull-secret.json
IMAGE_NAME=microshift-4.18-bootc

sudo modprobe openvswitch
sudo podman run --rm -it --privileged \
    -v "${PULL_SECRET}":/etc/crio/openshift-pull-secret:ro \
    -v /var/lib/containers/storage:/var/lib/containers/storage \
    --name "${IMAGE_NAME}" \
    "${IMAGE_NAME}"
```

> The `systemd-modules-load` service will fail to start in the container if the
> host kernel version is different from the `bootc` image kernel version. This
> failure can be safely ignored as all the necessary kernel modules have already
> been loaded by the host.

> If additional LVM volume group device was allocated as described in the
> [Configure CSI](#configure-csi) section, the loop device should automatically
> be shared with the container and used by the MicroShift CSI driver.

After the MicroShift `bootc` image has been successfully started, a login prompt
will be presented in the terminal. Log into the running container using the
`redhat:<password>` credentials.

Run the following command to verify that all the MicroShift pods are up and running
without errors.

```bash
watch sudo oc get pods -A \
    --kubeconfig /var/lib/microshift/resources/kubeadmin/kubeconfig
```

> Run the `sudo shutdown now` command to stop the container.

## Run MicroShift Bootc Virtual Machine

Log into the `physical hypervisor host` using the user credentials that have
SUDO permissions configured.

### Prepare Kickstart File

Set variables pointing to secret files that are included in `kickstart.ks` for
gaining access to private container registries:
* `AUTH_CONFIG` file contents are copied to `/etc/ostree/auth.json` at the
pre-install stage to authenticate `quay.io/myorg` registry access
* `PULL_SECRET` file contents are copied to `/etc/crio/openshift-pull-secret`
at the post-install stage to authenticate OpenShift registry access
* `IMAGE_REF` variable contains the MicroShift bootc container image reference
to be installed

```bash
AUTH_CONFIG=~/.quay-auth.json
PULL_SECRET=~/.pull-secret.json
IMAGE_REF="quay.io/<myorg>/<mypath>/microshift-4.18-bootc"
```

> See the `containers-auth.json(5)` manual pages for more information on the
> syntax of the `AUTH_CONFIG` registry authentication file.

Run the following commands to create the `kickstart.ks` file to be used during
the virtual machine installation.

```bash
cat > kickstart.ks <<EOFKS
lang en_US.UTF-8
keyboard us
timezone UTC
text
reboot

# Partition the disk with hardware-specific boot and swap partitions, adding an
# LVM volume that contains a 10GB+ system root. The remainder of the volume will
# be used by the CSI driver for storing data.
zerombr
clearpart --all --initlabel
# Create boot and swap partitions as required by the current hardware platform
reqpart --add-boot
# Add an LVM volume group and allocate a system root logical volume
part pv.01 --grow
volgroup rhel pv.01
logvol / --vgname=rhel --fstype=xfs --size=10240 --name=root

# Lock root user account
rootpw --lock

# Configure network to use DHCP and activate on boot
network --bootproto=dhcp --device=link --activate --onboot=on

%pre-install --log=/dev/console --erroronfail

# Create a 'bootc' image registry authentication file
mkdir -p /etc/ostree
cat > /etc/ostree/auth.json <<'EOF'
$(cat "${AUTH_CONFIG}")
EOF

%end

# Pull a 'bootc' image from a remote registry
ostreecontainer --url "${IMAGE_REF}"

%post --log=/dev/console --erroronfail

# Create an OpenShift pull secret file
cat > /etc/crio/openshift-pull-secret <<'EOF'
$(cat "${PULL_SECRET}")
EOF
chmod 600 /etc/crio/openshift-pull-secret

%end
EOFKS
```

The kickstart file uses a special [ostreecontainer](https://pykickstart.readthedocs.io/en/latest/kickstart-docs.html#ostreecontainer)
directive to pull a `bootc` image from the remote registry and use it to install
the RHEL operating system.

> Replace `myorg/mypath` with your remote registry organization name and path.

### Create Virtual Machine

Download a RHEL boot ISO image from https://developers.redhat.com/products/rhel/download.
Copy the downloaded file to the `/var/lib/libvirt/images` directory.

Run the following commands to create a RHEL virtual machine with 2 cores, 2GB of
RAM and 20GB of storage. The command uses the kickstart file prepared in the
previous step to pull a `bootc` image from the remote registry and use it to install
the RHEL operating system.

```bash
VMNAME=microshift-4.18-bootc
NETNAME=default

sudo virt-install \
    --name ${VMNAME} \
    --vcpus 2 \
    --memory 2048 \
    --disk path=/var/lib/libvirt/images/${VMNAME}.qcow2,size=20 \
    --network network=${NETNAME},model=virtio \
    --events on_reboot=restart \
    --location /var/lib/libvirt/images/rhel-9.4-$(uname -m)-boot.iso \
    --initrd-inject kickstart.ks \
    --extra-args "inst.ks=file://kickstart.ks" \
    --wait
```

Log into the virtual machine using the `redhat:<password>` credentials.
Run the following command to verify that all the MicroShift pods are up and running
without errors.

```bash
watch sudo oc get pods -A \
    --kubeconfig /var/lib/microshift/resources/kubeadmin/kubeconfig
```

## Using Bootc Image Builder (BIB)

The [bootc-image-builder](https://github.com/osbuild/bootc-image-builder), is a
containerized tool to create disk images from bootc images. You can use the images
that you build to deploy disk images in different environments, such as the edge,
server, and clouds.

### Create ISO image using BIB

```bash
PULL_SECRET=~/.pull-secret.json
IMAGE_NAME=microshift-4.18-bootc

mkdir ./output
sudo podman run --authfile ${PULL_SECRET} --rm -it \
    --privileged \
    --security-opt label=type:unconfined_t \
    -v /var/lib/containers/storage:/var/lib/containers/storage \
    -v ./output:/output \
    registry.redhat.io/rhel9/bootc-image-builder:latest \
    --local \
    --type iso \
    localhost/${IMAGE_NAME}:latest
```

### Prepare Kickstart File

Set variables pointing to secret files that are included in `kickstart.ks` for
gaining access to private container registries:
* `PULL_SECRET` file contents are copied to `/etc/crio/openshift-pull-secret`
  at the post-install stage to authenticate OpenShift registry access

```bash
PULL_SECRET=~/.pull-secret.json
```

Run the following command to create the `kickstart.ks` file to be used during
the virtual machine installation. If you want to embed the kickstart file directly
to iso using BIB please refer to [upstream docs](https://osbuild.org/docs/bootc/#anaconda-iso-installer-options-installer-mapping)

```bash
cat > kickstart.ks <<EOFKS
lang en_US.UTF-8
keyboard us
timezone UTC
text
reboot

# Partition the disk with hardware-specific boot and swap partitions, adding an
# LVM volume that contains a 10GB+ system root. The remainder of the volume will
# be used by the CSI driver for storing data.
zerombr
clearpart --all --initlabel
# Create boot and swap partitions as required by the current hardware platform
reqpart --add-boot
# Add an LVM volume group and allocate a system root logical volume
part pv.01 --grow
volgroup rhel pv.01
logvol / --vgname=rhel --fstype=xfs --size=10240 --name=root

# Lock root user account
rootpw --lock

# Configure network to use DHCP and activate on boot
network --bootproto=dhcp --device=link --activate --onboot=on

%post --log=/dev/console --erroronfail

# Create an OpenShift pull secret file
cat > /etc/crio/openshift-pull-secret <<'EOF'
$(cat "${PULL_SECRET}")
EOF
chmod 600 /etc/crio/openshift-pull-secret

%end
EOFKS
```

### Create Virtual Machine

Copy the `install.iso` file to the `/var/lib/libvirt/images` directory.

```bash
VMNAME=microshift-4.18-bootc
NETNAME=default

sudo cp -Z ./output/bootiso/install.iso /var/lib/libvirt/images/${VMNAME}.iso

sudo virt-install \
    --name ${VMNAME} \
    --vcpus 2 \
    --memory 2048 \
    --disk path=/var/lib/libvirt/images/${VMNAME}.qcow2,size=20 \
    --network network=${NETNAME},model=virtio \
    --events on_reboot=restart \
    --location /var/lib/libvirt/images/${VMNAME}.iso \
    --initrd-inject kickstart.ks \
    --extra-args "inst.ks=file://kickstart.ks" \
    --wait
```

Log into the virtual machine using the `redhat:<password>` credentials.
Run the following command to verify that all the MicroShift pods are up and running
without errors.

```bash
watch sudo oc get pods -A \
    --kubeconfig /var/lib/microshift/resources/kubeadmin/kubeconfig
```

## Appendix A: Multi-Architecture Image Build

It is often convenient to build multi-architecture container images and store
them under the same registry URL using manifest lists.

> See [podman-manifest](https://docs.podman.io/en/latest/markdown/podman-manifest.1.html) for more information.

The [Build Image](#build-image) procedure needs to be adjusted in the following
manner to create multi-architecture images.

```bash
PULL_SECRET=~/.pull-secret.json
USER_PASSWD="<your_redhat_user_password>"
IMAGE_ARCH=amd64 # Use amd64 or arm64 depending on the current platform
IMAGE_PLATFORM="linux/${IMAGE_ARCH}"
IMAGE_NAME="microshift-4.18-bootc:linux-${IMAGE_ARCH}"

sudo podman build --authfile "${PULL_SECRET}" -t "${IMAGE_NAME}" \
    --platform "${IMAGE_PLATFORM}" \
    --build-arg USER_PASSWD="${USER_PASSWD}" \
    -f Containerfile
```

Verify that the local MicroShift 4.18 `bootc` image was created for the specified
platform.

```bash
$ sudo podman images "${IMAGE_NAME}"
REPOSITORY                       TAG          IMAGE ID      CREATED         SIZE
localhost/microshift-4.18-bootc  linux-amd64  3f7e136fccb5  13 minutes ago  2.19 GB
```

Repeat the procedure on the other platform (i.e. `arm64`) and proceed by publishing
the platform-specific `amd64` and `arm64` images to the remote registry as described
in the [Publish Image](#publish-image) section.

> Cross-platform `podman` builds are not in the scope of this document. Log into
> the RHEL 9.4 host running on the appropriate architecture to perform the container
> image builds and publish the platform-specific image to the remote registry.

Finally, create a manifest containing the platform-specific image references
and publish it to the remote registry.

> Images for **both** `amd64` and `arm64` architectures should have been pushed
> to the remote registry before creating the manifest.

```bash
REGISTRY_URL=quay.io
REGISTRY_ORG=myorg/mypath
BASE_NAME=microshift-4.18-bootc
MANIFEST_NAME="${BASE_NAME}:latest"

sudo podman manifest create -a "localhost/${MANIFEST_NAME}" \
    "${REGISTRY_URL}/${REGISTRY_ORG}/${BASE_NAME}:linux-amd64" \
    "${REGISTRY_URL}/${REGISTRY_ORG}/${BASE_NAME}:linux-arm64"

sudo podman manifest push \
    "localhost/${MANIFEST_NAME}" \
    "${REGISTRY_URL}/${REGISTRY_ORG}/${MANIFEST_NAME}"
```

> Replace `myorg/mypath` with your remote registry organization name and path.

Inspect the remote manifest to make sure it contains image digests from multiple
architectures.

```bash
$ sudo podman manifest inspect \
    "${REGISTRY_URL}/${REGISTRY_ORG}/${MANIFEST_NAME}" | \
    jq .manifests[].platform.architecture
"amd64"
"arm64"
```

It is now possible to access images using the manifest name with the `latest` tag
(e.g. `quay.io/myorg/mypath/microshift-4.18-bootc:latest`). The image for the
current platform will automatically be pulled from the registry if it is part of
the manifest list.

## Appendix B: Embedding Container Images in Bootc Builds

Adding MicroShift container image dependencies to bootc images may be necessary
for isolated (no Internet access) setup or for improving MicroShift first startup
performance. The container image references are specific to platform and to each
MicroShift version.

### Build Container Image

Download the [Containerfile.embedded](../config/Containerfile.bootc-embedded-rhel9) using
the following command and use it for subsequent image builds.

```bash
URL=https://raw.githubusercontent.com/openshift/microshift/refs/heads/main/docs/config/Containerfile.bootc-embedded-rhel9

curl -s -o Containerfile.embedded "${URL}"
```

Run the following image build command to create a local `bootc` image with embedded
container dependencies. It is using a base image built according to the instructions
in the [Build Image](#build-image) section.

Note how secrets are used during the image build:
* The podman `--authfile` argument is required to pull the base image from the
`registry.redhat.io` registry
* The podman `--secret` argument is required to pull image dependencies from the
OpenShift container registries.

```bash
PULL_SECRET=~/.pull-secret.json
BASE_IMAGE_NAME=microshift-4.18-bootc
BASE_IMAGE_TAG=latest
IMAGE_NAME=microshift-4.18-bootc-embedded

sudo podman build --authfile "${PULL_SECRET}" -t "${IMAGE_NAME}" \
    --secret "id=pullsecret,src=${PULL_SECRET}" \
    --build-arg USHIFT_BASE_IMAGE_NAME="${BASE_IMAGE_NAME}" \
    --build-arg USHIFT_BASE_IMAGE_TAG="${BASE_IMAGE_TAG}" \
    -f Containerfile.embedded
```

Verify that the local MicroShift 4.18 `bootc` image was created.

```bash
$ sudo podman images "${IMAGE_NAME}"
REPOSITORY                                TAG         IMAGE ID      CREATED             SIZE
localhost/microshift-4.18-bootc-embedded  latest      6490d8f5752a  About a minute ago  3.75 GB
```

### Build Installation Image

Follow the instructions in [Create ISO Image Using BIB](#create-iso-image-using-bib)
to build an ISO from the container image with embedded container dependencies.

> Note: Make sure to set the `IMAGE_NAME` variable to `microshift-4.18-bootc-embedded`

### Prepare Kickstart File

Set variables pointing to secret files that are included in `kickstart.ks` for
gaining access to private container registries:
* `PULL_SECRET` file contents are copied to `/etc/crio/openshift-pull-secret`
  at the post-install stage to authenticate OpenShift registry access

```bash
PULL_SECRET=~/.pull-secret.json
IMAGE_NAME=microshift-4.18-bootc-embedded
```

Run the following command to create the `kickstart.ks` file to be used during
the virtual machine installation.

```bash
cat > kickstart.ks <<EOFKS
lang en_US.UTF-8
keyboard us
timezone UTC
text
reboot

# Partition the disk with hardware-specific boot and swap partitions, adding an
# LVM volume that contains a 10GB+ system root. The remainder of the volume will
# be used by the CSI driver for storing data.
zerombr
clearpart --all --initlabel
# Create boot and swap partitions as required by the current hardware platform
reqpart --add-boot
# Add an LVM volume group and allocate a system root logical volume
part pv.01 --grow
volgroup rhel pv.01
logvol / --vgname=rhel --fstype=xfs --size=10240 --name=root

# Lock root user account
rootpw --lock

# Configure network to use DHCP and activate on boot
network --bootproto=dhcp --device=link --activate --onboot=on

# Configure bootc to install from the local embedded container repository.
# See /osbuild-base.ks on ISO images generated by bootc-image-builder.
ostreecontainer --transport oci --url /run/install/repo/container

%post --log=/dev/console --erroronfail

# Update the image reference for updates to work correctly.
# See /osbuild.ks on ISO images generated by bootc-image-builder.
bootc switch --mutate-in-place --transport registry localhost/${IMAGE_NAME}

# Create an OpenShift pull secret file
cat > /etc/crio/openshift-pull-secret <<'EOF'
$(cat "${PULL_SECRET}")
EOF
chmod 600 /etc/crio/openshift-pull-secret

%end
EOFKS
```

### Configure Isolated Network

Before creating a virtual machine, it is necessary to configure a `libvirt`
network without Internet access. Run the following commands to create such
a network.

```bash
VM_ISOLATED_NETWORK=microshift-isolated-network

cat > isolated-network.xml <<EOF
<network>
  <name>${VM_ISOLATED_NETWORK}</name>
  <forward mode='none'/>
  <ip address='192.168.111.1' netmask='255.255.255.0' localPtr='yes'>
    <dhcp>
      <range start='192.168.111.100' end='192.168.111.254'/>
    </dhcp>
  </ip>
</network>
EOF

sudo virsh net-define isolated-network.xml
sudo virsh net-start     "${VM_ISOLATED_NETWORK}"
sudo virsh net-autostart "${VM_ISOLATED_NETWORK}"
```

### Create Virtual Machine

Follow the instructions in [Create Virtual Machine](#create-virtual-machine-1)
to bootstrap a virtual machine from the ISO with embedded container dependencies.

> Note: Make sure to set the `NETNAME` variable to the `VM_ISOLATED_NETWORK`
> isolated network name.

Log into the virtual machine **console** using the `redhat:<password>` credentials.

Run the following command to verify that there is no Internet access, thus
no container image dependencies could have been pulled over the network.

```bash
$ curl -I redhat.com
curl: (6) Could not resolve host: redhat.com
```

Run the following command to verify that all the MicroShift pods are up and running
without errors.

```bash
watch sudo oc get pods -A \
    --kubeconfig /var/lib/microshift/resources/kubeadmin/kubeconfig
```

## Appendix C: The rpm-ostree to Image Mode Upgrade Procedure

Refer to RHEL documentation for generic instructions on upgrading `rpm-ostree`
systems to Image Mode. The upgrade process should be planned carefully considering
the following guidelines:
* Follow instructions in RHEL documentation for converting `rpm-ostree` blueprints to
  Image Mode container files
* Consider using [rpm-ostree compose container-encapsulate](https://coreos.github.io/rpm-ostree/container/#converting-ostree-commits-to-new-base-images)
  to experiment with Image Mode based on the existing `ostree` commits
* Invest in defining a proper container build pipeline for fully adopting Image Mode

If reinstalling MicroShift devices from scratch is not an option, read the remainder
of this section that outlines the upgrade details specific to MicroShift.

Upgrading existing systems during the transition from `rpm-ostree` to Image Mode
may pose the challenge of [UID / GID Drift](https://github.com/bootc-dev/bootc/issues/673)
because the existing `rpm-ostree` and the new Image Mode images are not derived
from the same parent image.

One way of working around this issue is to add `systemd` units that run before the
affected system services and apply the necessary fixes.

> Note: The workaround is only necessary for `rpm-ostree` to Image Mode upgrade
> and it can be removed once all the devices are running the upgraded image.

Add the following command to the MicroShift image build procedure to create a
`systemd` unit file solving a potential UID / GID drift for `ovsdb-server.service`.

```
# Install systemd configuration drop-ins to fix potential permission problems
# when upgrading from older rpm-ostree commits to Image Mode container layers
RUN mkdir -p /usr/lib/systemd/system/ovsdb-server.service.d && \
    cat > /usr/lib/systemd/system/ovsdb-server.service.d/microshift-ovsdb-ownership.conf <<'EOF'
# The openvswitch database files must be owned by the appropriate user and its
# primary group. Note that the user and its group may be overwritten too, so
# they need to be recreated in this case.
[Service]
ExecStartPre=/bin/sh -c '/bin/getent passwd openvswitch >/dev/null || useradd -r openvswitch'
ExecStartPre=/bin/sh -c '/bin/getent group hugetlbfs >/dev/null || groupadd -r hugetlbfs'
ExecStartPre=/sbin/usermod -a -G hugetlbfs openvswitch
ExecStartPre=/bin/chown -Rhv openvswitch. /etc/openvswitch
EOF
