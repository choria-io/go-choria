%define debug_package %{nil}
%define pkgname {{cpkg_name}}
%define version {{cpkg_version}}
%define bindir {{cpkg_bindir}}
%define etcdir {{cpkg_etcdir}}
%define release {{cpkg_release}}
%define dist {{cpkg_dist}}
%define manage_conf {{cpkg_manage_conf}}
%define binary {{cpkg_binary}}
%define tarball {{cpkg_tarball}}
%define contact {{cpkg_contact}}
%define pkggroup {{cpkg_rpm_group}}

Name: %{pkgname}
Version: %{version}
Release: %{release}.%{dist}
Summary: The Choria Orchestrator Server
License: Apache-2.0
URL: https://choria.io
Group: %{pkggroup}
Source0: %{tarball}
Packager: %{contact}
BuildRoot: %{_tmppath}/%{pkgname}-%{version}-%{release}-root-%(%{__id_u} -n)

%description
The Choria Orchestrator Server and Broker

Please visit https://choria.io for more information

%prep
%setup -q

%build

%install
rm -rf %{buildroot}
%{__install} -d -m0755  %{buildroot}/etc/sysconfig
%{__install} -d -m0755  %{buildroot}/etc/logrotate.d
%{__install} -d -m0755  %{buildroot}/etc/rc.d/init.d
%{__install} -d -m0755  %{buildroot}%{bindir}
%{__install} -d -m0755  %{buildroot}%{etcdir}
%{__install} -d -m0755  %{buildroot}/var/log
%{__install} -m0755 dist/server.init %{buildroot}/etc/rc.d/init.d/%{pkgname}-server
%{__install} -m0755 dist/broker.init %{buildroot}/etc/rc.d/init.d/%{pkgname}-broker
%{__install} -m0644 dist/server.sysconfig %{buildroot}/etc/sysconfig/%{pkgname}-server
%{__install} -m0644 dist/broker.sysconfig %{buildroot}/etc/sysconfig/%{pkgname}-broker
%{__install} -m0755 dist/choria-logrotate %{buildroot}/etc/logrotate.d/%{pkgname}
%if 0%{?manage_conf} > 0
%{__install} -m0640 dist/server.conf %{buildroot}%{etcdir}/server.conf
%{__install} -m0640 dist/broker.conf %{buildroot}%{etcdir}/broker.conf
%endif
%{__install} -m0755 %{binary} %{buildroot}%{bindir}/%{pkgname}

%clean
rm -rf %{buildroot}

%post
/sbin/chkconfig --add %{pkgname}-broker || :
/sbin/chkconfig --add %{pkgname}-server || :

%postun
if [ "$1" -ge 1 ]; then
  /sbin/service %{pkgname}-broker condrestart &>/dev/null || :
  /sbin/service %{pkgname}-server condrestart &>/dev/null || :
fi

%preun
if [ "$1" = 0 ] ; then
  /sbin/service %{pkgname}-broker stop > /dev/null 2>&1
  /sbin/chkconfig --del %{pkgname}-broker || :
  /sbin/service %{pkgname}-server stop > /dev/null 2>&1
  /sbin/chkconfig --del %{pkgname}-server || :
fi

%files
%if 0%{?manage_conf} > 0
%config(noreplace)%{etcdir}/broker.conf
%config(noreplace)%{etcdir}/server.conf
%endif
%{bindir}/%{pkgname}
/etc/logrotate.d/%{pkgname}
/etc/rc.d/init.d/%{pkgname}-server
%config(noreplace)/etc/sysconfig/%{pkgname}-server
/etc/rc.d/init.d/%{pkgname}-broker
%config(noreplace)/etc/sysconfig/%{pkgname}-broker

%changelog
* Tue Dec 05 2017 R.I.Pienaar <rip@devco.net>
- Initial Release
