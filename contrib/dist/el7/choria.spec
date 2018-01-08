%define debug_package %{nil}
%define pkgname {{pkgname}}
%define version {{version}}
%define bindir {{bindir}}
%define etcdir {{etcdir}}
%define iteration {{iteration}}
%define dist {{dist}}
%define manage_conf {{manage_conf}}
%define target_dist {{target_dist}}
%define target_arch {{target_arch}}
%define contact {{contact}}

Name: %{pkgname}
Version: %{version}
Release: %{iteration}.%{target_dist}
Summary: The Choria Orchestrator Server
License: Apache-2.0
URL: https://choria.io
Group: System Tools
Packager: %{contact}
Source0: %{pkgname}-%{version}-linux-%{target_arch}.tgz
BuildRoot: %{_tmppath}/%{pkgname}-%{version}-%{release}-root-%(%{__id_u} -n)

%description
The Choria Orchestrator Server and Broker

%prep
%setup -q

%build

%install
rm -rf %{buildroot}
%{__install} -d -m0755  %{buildroot}/usr/lib/systemd/system
%{__install} -d -m0755  %{buildroot}/etc/logrotate.d
%{__install} -d -m0755  %{buildroot}%{bindir}
%{__install} -d -m0755  %{buildroot}%{etcdir}
%{__install} -d -m0755  %{buildroot}/var/log
%{__install} -m0644 dist/server.service %{buildroot}/usr/lib/systemd/system/%{pkgname}-server.service
%{__install} -m0644 dist/broker.service %{buildroot}/usr/lib/systemd/system/%{pkgname}-broker.service
%{__install} -m0644 dist/choria-logrotate %{buildroot}/etc/logrotate.d/%{pkgname}
%if 0%{?manage_conf} > 0
%{__install} -m0640 dist/server.conf %{buildroot}%{etcdir}/server.conf
%{__install} -m0640 dist/broker.conf %{buildroot}%{etcdir}/broker.conf
%endif
%{__install} -m0755 choria-%{version}-linux-%{target_arch} %{buildroot}%{bindir}/%{pkgname}

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


%changelog
* Tue Dec 05 2017 R.I.Pienaar <rip@devco.net>
- Initial Release

