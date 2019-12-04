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
Packager: %{contact}
Source0: %{tarball}
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
%{__install} -d -m0755  %{buildroot}/usr/lib/systemd/system
%{__install} -d -m0755  %{buildroot}/etc/logrotate.d
%{__install} -d -m0755  %{buildroot}%{bindir}
%{__install} -d -m0755  %{buildroot}%{etcdir}
%{__install} -d -m0755  %{buildroot}/var/log
%{__install} -m0644 dist/server.sysconfig %{buildroot}/etc/sysconfig/%{pkgname}-server
%{__install} -m0644 dist/server.service %{buildroot}/usr/lib/systemd/system/%{pkgname}-server.service
%{__install} -m0644 dist/broker.service %{buildroot}/usr/lib/systemd/system/%{pkgname}-broker.service
%{__install} -m0644 dist/choria-logrotate %{buildroot}/etc/logrotate.d/%{pkgname}
%if 0%{?manage_conf} > 0
%{__install} -m0640 dist/server.conf %{buildroot}%{etcdir}/server.conf
%{__install} -m0640 dist/broker.conf %{buildroot}%{etcdir}/broker.conf
%endif
%{__install} -m0755 %{binary} %{buildroot}%{bindir}/%{pkgname}

%clean
rm -rf %{buildroot}

%post
if [ $1 -eq 1 ] ; then
  systemctl --no-reload preset %{pkgname}-broker >/dev/null 2>&1 || :
  systemctl --no-reload preset %{pkgname}-server >/dev/null 2>&1 || :
fi

/bin/systemctl --system daemon-reload >/dev/null 2>&1 || :

if [ $1 -ge 1 ]; then
  /bin/systemctl try-restart %{pkgname}-broker >/dev/null 2>&1 || :;
  /bin/systemctl try-restart %{pkgname}-server >/dev/null 2>&1 || :;
fi

%preun
if [ $1 -eq 0 ] ; then
  systemctl --no-reload disable --now %{pkgname}-broker >/dev/null 2>&1 || :
  systemctl --no-reload disable --now %{pkgname}-server >/dev/null 2>&1 || :
fi

%files
%if 0%{?manage_conf} > 0
%config(noreplace)%{etcdir}/broker.conf
%config(noreplace)%{etcdir}/server.conf
%endif
%{bindir}/%{pkgname}
/etc/logrotate.d/%{pkgname}
/usr/lib/systemd/system/%{pkgname}-server.service
/usr/lib/systemd/system/%{pkgname}-broker.service
%config(noreplace)/etc/sysconfig/%{pkgname}-server

%changelog
* Tue Dec 05 2017 R.I.Pienaar <rip@devco.net>
- Initial Release

