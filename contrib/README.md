# Building RPMS

First build the binary, this cannot be cross compiled in real world so compile it on some linux machine:

```
VERSION=0.0.1myco rake build
```

This will give you a binary like `choria-0.0.1myco-Linux-amd64`, put it in this directory.

Now you need to build the Docker image that will make the RPMs.  This uses OEL5 to ensure compatibility with old distros, something is currently wrong with CentOS 5 docker images

```
docker build . --tag rpmbuilder
```

You can now build your package, this shows how to customise paths, names etc.  Dist can be either el6 or el7:

```
docker run -v `pwd`:/build \
  -e VERSION=0.0.1 \
  -e RELEASE=myco \
  -e DIST=el7 \
  -e NAME=myco-choria \
  -e BINDIR=/opt/myco/choria/bin \
  -e ETCDIR=/opt/myco/choria/etc \
  -e ITERATION=2 \
  --rm rpmbuilder
```

At the end you'll have:

```
-rw-r--r-- 1 root root  4661742 Dec  6 12:25 myco-choria-0.0.1-myco-Linux-amd64.tgz
-rw-r--r-- 1 root root     3891 Dec  6 12:25 myco-choria-broker-0.0.1-6myco.el7.x86_64.rpm
-rw-r--r-- 1 root root  3599095 Dec  6 12:25 myco-choria-0.0.1-2myco.el7.x86_64.rpm
-rw-r--r-- 1 root root  4649581 Dec  6 12:25 myco-choria-0.0.1-2myco.el7.src.rpm
```

The binaries, logs, services, etc will all reflect your chosen name
