# Building RPMS

First build the binary, this cannot be cross compiled in real world so compile it on some linux machine:

If you're new to building Go stuff then get go on your machine:

```
# visit golang.org and get the linux tarball
tar -C /usr/local -xzf go1.9.2.linux-amd64.tar.gz

yum -y install git rubygem-rake
export GOPATH=~/go
PATH=$PATH:/usr/local/go/bin:~/go/bin

go get github.com/Masterminds/glide
mkdir -p ${GOPATH}/src/github.com/choria-io
cd ${GOPATH}/src/github.com/choria-io
git clone https://github.com/choria-io/go-choria.git
cd go-choria
glide install
```

```
VERSION=0.0.1myco rake build
./choria-0.0.1myco-Linux-amd64  --version
version: 0.0.1myco

license: Apache-2.0
built: 2017-12-06 11:52:13 +0000
sha: 04f46f3
tls: true
secure: true
go: go1.9.2
```

This will give you a binary like `choria-0.0.1myco-Linux-amd64`, put it in this directory.

Now you need to build the Docker image that will make the RPMs.  This uses OEL5 to ensure compatibility with old distros, something is currently wrong with CentOS 5 docker images

```
docker build . --tag rpmbuilder
```

You can now build your package, this shows how to customise paths, names etc.  Dist can be either el6 or el7:

```
docker run -v `pwd`:/build \
  -e VERSION=0.0.1myco \
  -e RELEASE=2 \
  -e DIST=el7 \                    # uses config/spec etc in dist/el7
  -e NAME=myco-choria \
  -e BINDIR=/opt/myco/choria/bin \
  -e ETCDIR=/opt/myco/choria/etc \
  -e MANAGE_CONF=1 \               # set to zero to not add config files to the package
  --rm rpmbuilder
```

At the end you'll have:

```
-rw-r--r-- 1 root root     3891 Dec  6 12:25 myco-choria-broker-0.0.1-2myco.el7.x86_64.rpm
-rw-r--r-- 1 root root  3599095 Dec  6 12:25 myco-choria-0.0.1-2myco.el7.x86_64.rpm
-rw-r--r-- 1 root root  4649581 Dec  6 12:25 myco-choria-0.0.1-2myco.el7.src.rpm
```

The binaries, logs, services, etc will all reflect your chosen name.
