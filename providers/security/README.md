# Choria Security Providers

This provides a unified interface to PKI systems that all the Choria eco system projects can use to present a more unified UI / UX.

## Providers

At present there are only 2 providers - `file` and `puppet` - in future we'll support a Choria specific CA and others like Vault and perhaps those provided by public Clouds.

|Provider|Description|
|--------|-----------|
|Puppet  |Understands the structure of SSL files maintained by `puppet agent`, supports enrolling into a PuppetCA|
|File    |Accepts a fully manual configuration with paths to all the major needed files, does not support enrollment|

## CLI

You can do arbitrary enrolls using the CLI provided here:

```
$ pki-enroll --help
usage: pki-enroll [<flags>] <identity>

Enrolls with various PKI systems using the Choria framework

Flags:
  --help                     Show context-sensitive help (also try --help-long and --help-man).
  --version                  Show application version.
  --scheme=puppet            Provider to enroll with, only support 'puppet'
  --wait=30m                 How long to wait for the certificate to be signed
  --puppet-ssldir=PATH       The directory to write the Puppet compatible SSL structure
  --puppet-ca="puppet:8140"  PuppetCA in host:port format
  --verbose                  Verbose logging

Args:
  <identity>  Identity to enroll as
```

Enrolling into a PuppetCA entails the following:

  * Create a private key
  * Create a CSR
  * Download the CA
  * Submit the CSR
  * Repeatedly attempt to download the signed certificate until someone issues `puppet cert sign` on the CA

Here we use the `pki-enroll` command to perform this task with the resulting SSL tree created in `/tmp/ssl`.

```
$ pki-enroll bob --puppet-ssldir /tmp/ssl
Attempting to download certificate for bob, try 1.
Attempting to download certificate for bob, try 2.
Attempting to download certificate for bob, try 3.
```