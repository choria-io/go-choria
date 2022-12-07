+++
title = "Custom Packaging"
toc = true
weight = 60
pre = "<b>6. </b>"
+++

Choria binary, being a compiled binary with no external dependencies, needs to be recompiled when adding certain kinds
of plugin, changing some default locations or adding your own plugins.

The project provides the tooling to perform these builds and has a compile-time configuration that can be adjusted to local
needs.

### Requirements

The host used to perform the compile need to have Docker on it and be able to fetch the `choria/packager` container. You can
build a local version of the Packager using [https://github.com/choria-io/packager](https://github.com/choria-io/packager).

In general you should only do this if you know what you are doing, have special needs, want custom agents etc.

### Plugins

A number of [plugins types are supported](https://choria.io/docs/development/server/) and we build many in at compile time ourselves.

The general process is that all the plugins in `packager/plugins.yaml` will be included in the build, if you want to add
additional plugins you list them in `packager/user_plugins.yaml`.

If you wish to remove some default plugins you need to remove them from the `packager/plugin.yaml`.

In order to add your own RPC Agent you would list it in `packager/user_plugins.yaml`:

```yaml
---
acme: ghe.example.net/backplane/acme_agent
```

During your CI run `go get ghe.example.net/backplane/acme_agent` then `go generate` and start the build:

```nohighlight
BUILD=foss VERSION=0.26.0acme rake build
```

Your plugin will now be included in the final build, see `choria buildinfo -D` for al ist of all
dependencies, which should include your plugin.

### Custom builds, paths and packages

The `choria-io/go-choria` repository has `packager/buildspec.yaml` in it, this defines the binaries and packages to build,
there are also some supporting files to call RPM, Deb etc.

Lets look at building a custom 64bit Linux binary with different paths and creating an Enterprise Linux 8 RPM.

```yaml
flags_map:
  Version: github.com/choria-io/go-choria/build.Version
  SHA: github.com/choria-io/go-choria/build.SHA
  BuildTime: github.com/choria-io/go-choria/build.BuildDate
  ProvisionJWTFile: github.com/choria-io/go-choria/build.ProvisionJWTFile

acme:
  compile_targets:
    defaults:
      output: backplane-{{version}}-{{os}}-{{arch}}
      flags:
        ProvisionJWTFile: /etc/acme/backplane/provisioning.jwt
      pre:
        - rm additional_agent_*.go || true
        - rm plugin_*.go || true
        - go generate --run plugin

    64bit_linux:
      os: linux
      arch: amd64

  packages:
    defaults:
      name: backplane
      display_name: Backplane
      bindir: /opt/acme/bin
      etcdir: /etc/acme/backplane
      release: 1
      manage_conf: 1
      manage_server_preset: 0
      contact: Backplane Engineering <backplane@eng.example.com>
      rpm_group: System Environment/Base
      server_start_runlevels: "-"
      server_start_order: 50
      broker_start_runlevels: "-"
      broker_start_order: 50

    el8_64:
      template: el/el8
      dist: el8
      target_arch: x86_64
      binary: 64bit_linux
```

We can now run:

```nohighlight
BUILD=acme VERSION=0.26.0acme rake build
```

When you are done you will have:

 * an rpm called `backplane-0.26.0acme.el8.x86_64.rpm`
 * the binary will be `/opt/acme/bin/backplane`
 * config files, log files, services all will be personalized around `backplane`
 * it will have a custom path to the `provisioning.jwt`

A number of things are customizable see the section at the top of the `buildspec.yaml` and comments in the build file.
