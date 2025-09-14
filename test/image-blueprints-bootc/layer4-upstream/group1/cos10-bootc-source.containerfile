# WORKAROUND: Due to the lack of containernetworking-plugins package in CentOS Stream 10,
# we build a dummy package that satisfies the dependency.
FROM quay.io/centos-bootc/centos-bootc:stream10 AS builder

RUN cat > /tmp/dummy.spec <<'EOF'
Name:           containernetworking-plugins
Version:        2.0.0
Release:        1%{?dist}
Summary:        Dummy package to satisfy dependencies
License:        Public-Domain

Provides:       containernetworking-plugins = %{version}-%{release}

%description
Dummy package. Satisfies dependency on containernetworking-plugins.

%files
%doc
EOF
    
RUN pushd /tmp && \
    dnf install -y rpm-build && \
    rpmbuild -bb dummy.spec \
        --define "_rpmdir $(pwd)" \
        --define "_srcrpmdir $(pwd)" \
        --define "_builddir $(pwd)" \
        --define "_specdir $(pwd)" \
        --define "_sourcedir $(pwd)"

FROM quay.io/centos-bootc/centos-bootc:stream10

# Build arguments
ARG USHIFT_RPM_REPO_NAME=microshift-local
ARG USHIFT_RPM_REPO_PATH=/tmp/$USHIFT_RPM_REPO_NAME

# Copy the MicroShift repository contents
COPY ./rpm-repos/$USHIFT_RPM_REPO_NAME $USHIFT_RPM_REPO_PATH

# Copy repository configuration
COPY ./bootc-images/$USHIFT_RPM_REPO_NAME.repo ./bootc-images/microshift-centos10-nfv.repo ./bootc-images/microshift-rhocp-y.repo \
    /etc/yum.repos.d/

# WORKAROUND: Install containernetworking-plugins dummy package
# Note that the RPM package is located in the arch-specific subdirectory.
COPY --from=builder "/tmp/*/containernetworking-plugins-*.rpm" /tmp/
RUN rpm -ivh "/tmp/containernetworking-plugins-*.rpm" && \
    rm -vf "/tmp/containernetworking-plugins-*.rpm"

# Print repository configuration contents.
# Install MicroShift and cleanup.
RUN dnf repoinfo --enabled && \
    dnf install -y firewalld systemd-resolved \
        {{ range (env.Getenv "MICROSHIFT_MANDATORY_RPMS" | strings.Split " ") -}}
        "{{ . }}-{{ env.Getenv "SOURCE_VERSION" }}" \
        {{ end -}}
        && \
    systemctl enable microshift && \
    rm -vf /etc/yum.repos.d/microshift-*.repo && \
    rm -rvf $USHIFT_RPM_REPO_PATH && \
    dnf clean all

# Configure firewall
RUN firewall-offline-cmd --zone=public --add-port=22/tcp && \
    firewall-offline-cmd --zone=trusted --add-source=10.42.0.0/16 && \
    firewall-offline-cmd --zone=trusted --add-source=169.254.169.1 && \
    firewall-offline-cmd --zone=trusted --add-source=fd01::/48 && \
    firewall-offline-cmd --zone=public --add-port=80/tcp && \
    firewall-offline-cmd --zone=public --add-port=443/tcp && \
    firewall-offline-cmd --zone=public --add-port=5353/udp && \
    firewall-offline-cmd --zone=public --add-port=6443/tcp && \
    firewall-offline-cmd --zone=public --add-port=30000-32767/tcp && \
    firewall-offline-cmd --zone=public --add-port=30000-32767/udp
