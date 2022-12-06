+++
title = "Installation"
toc = true
weight = 30
pre = "<b>3. </b>"
+++

We distribute an RPM, Debs, MSIs, Exes and Docker containers for the Choria Server and Broker.

## Enterprise Linux

We publish RPM releases but also nightly builds to our repositories.

Users of our Puppet modules will already have these repositories available.

### Release

```ini
[choria_release]
name=Choria Orchestrator Releases
mirrorlist=http://mirrorlists.choria.io/yum/release/el/$releasever/$basearch.txt
enabled=True
gpgcheck=True
repo_gpgcheck=True
gpgkey=https://choria.io/RELEASE-GPG-KEY
metadata_expire=300
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
sslverify=True
```

### Nightly

Nightly releases are named and versioned `choria-0.99.0.20221109-1.el7.x86_64.rpm` where the last part of the version is the date.

```ini
[choria_nightly]
name=Choria Orchestrator Nightly
mirrorlist=http://mirrorlists.choria.io//yum/nightly/el/$releasever/$basearch.txt
enabled=True
gpgcheck=True
repo_gpgcheck=True
gpgkey=https://choria.io/NIGHTLY-GPG-KEY
metadata_expire=300
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
sslverify=True
```

## Debian

We publish release packages for Debian systems on our APT repositories:

```nohighlight
deb mirror://mirrorlists.choria.io/apt/release/debian/bullseye/mirrors.txt debian bullseye
```

## Docker

There is a docker container `choria-io/choria` that has releases only.
