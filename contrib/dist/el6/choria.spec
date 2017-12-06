%define  debug_package %{nil}

Name: %{pkgname}
Version: %{version}
Release: %{iteration}.%{dist}
Summary: The Choria Orchestrator Server
License: Apache-2.0
URL: https://choria.io
Group: System Tools
Source0: %{pkgname}-%{version}-Linux-amd64.tgz
Packager: R.I.Pienaar <rip@devco.net>
BuildRoot: %{_tmppath}/%{pkgname}-%{version}-%{release}-root-%(%{__id_u} -n)

%description
The Choria Orchestrator Server and Broker

%prep
%setup -q

%build
for i in server.init broker.init choria.conf choria-logrotate; do
  sed -i 's!{{pkgname}}!%{pkgname}!' dist/${i}
  sed -i 's!{{bindir}}!%{bindir}!' dist/${i}
  sed -i 's!{{etcdir}}!%{etcdir}!' dist/${i}
done

%install
rm -rf %{buildroot}
%{__install} -d -m0755  %{buildroot}/etc/sysconfig
%{__install} -d -m0755  %{buildroot}/etc/init.d
%{__install} -d -m0755  %{buildroot}/etc/logrotate.d
%{__install} -d -m0755  %{buildroot}%{bindir}
%{__install} -d -m0755  %{buildroot}%{etcdir}
%{__install} -d -m0755  %{buildroot}/var/log
%{__install} -m0755 dist/server.init %{buildroot}/etc/init.d/%{pkgname}-server
%{__install} -m0755 dist/broker.init %{buildroot}/etc/init.d/%{pkgname}-broker
%{__install} -m0644 dist/server.sysconfig %{buildroot}/etc/sysconfig/%{pkgname}-server
%{__install} -m0644 dist/broker.sysconfig %{buildroot}/etc/sysconfig/%{pkgname}-broker
%{__install} -m0755 dist/choria-logrotate %{buildroot}/etc/logrotate.d/%{pkgname}
%if 0%{?manage_conf} > 0
%{__install} -m0640 dist/choria.conf %{buildroot}%{etcdir}/choria.conf
%endif
%{__install} -m0755 choria-%{version}-Linux-amd64 %{buildroot}%{bindir}/%{pkgname}

%clean
rm -rf %{buildroot}

%post
/sbin/chkconfig --add %{pkgname}-broker || :
/sbin/chkconfig --add %{pkgname}-server || :

%postun
if [ "$1" -ge 1 ]; then
  /sbin/service %{pkgname}-broker condrestart &>/dev/null || :
fi

if [ "$1" -ge 1 ]; then
  /sbin/service %{pkgname}-server condrestart &>/dev/null || :
fi

%preun
if [ "$1" = 0 ] ; then
  /sbin/service %{pkgname}-broker stop > /dev/null 2>&1
  /sbin/chkconfig --del %{pkgname}-broker || :
fi

if [ "$1" = 0 ] ; then
  /sbin/service %{pkgname}-server stop > /dev/null 2>&1
  /sbin/chkconfig --del %{pkgname}-server || :
fi

%files
%if 0%{?manage_conf} > 0
%config(noreplace)%{etcdir}/choria.conf
%endif
%{bindir}/%{pkgname}
/etc/logrotate.d/%{pkgname}
/etc/init.d/%{pkgname}-server
/etc/sysconfig/%{pkgname}-server
/etc/init.d/%{pkgname}-broker
/etc/sysconfig/%{pkgname}-broker

%changelog
* Tue Dec 05 2017 R.I.Pienaar <rip@devco.net>
- Initial Release
