%global min_golang_ver 1.16

%if "%{_vendor}" == "debbuild"
 # Force bash for shell for sanity
%global _buildshell /bin/bash
%endif

Name:     sisyphus
Version:  0.1.6
Release:  1%{?dist}
Summary:  Streaming data from Kafka to InfluxDB v2 endpoints
License:  GPLv3+
URL:      https://github.com/datto/sisyphus
Source0:  %{url}/archive/%{version}/%{name}-%{version}.tar.gz
%if "%{_vendor}" == "debbuild"
Group:   misc
Packager: John Seekins <jseekins@datto.com>
%else
Group:    Applications/Text
%endif

%{?systemd_requires}
BuildRequires: systemd
%if "%{_vendor}" == "debbuild"
BuildRequires:  golang >= 3:%{min_golang_ver}
BuildRequires:  go-deb-macros
BuildRequires: systemd-deb-macros
%else
BuildRequires:  golang >= %{min_golang_ver}
BuildRequires: systemd-rpm-macros
%endif

%description
Kafka -> Influx 2.x forwarder in Golang
Handles formatting data per the Prometheus data model as well.

%prep
%autosetup -p1

%build
# If SOURCE_DATE_EPOCH isn't set, set it
if [ -z "$SOURCE_DATE_EPOCH" ]; then
    if [ ! -f "%{_specdir}/%{name}.spec" ]; then
        # OBS does not install the spec file into the specdir
        SPECFILE_PATH="%{_sourcedir}/%{name}.spec"
    else
        SPECFILE_PATH="%{_specdir}/%{name}.spec"
    fi
    export SOURCE_DATE_EPOCH=$(stat --printf='%Y' ${SPECFILE_PATH})
fi

# Set the commit as part of the build
export LDFLAGS="-X 'main.Version=%{version}' -X 'main.Builder=OBS' -X 'main.BuildTime=${SOURCE_DATE_EPOCH}'"

# Turn go modules on
export GO111MODULE='on'

go build -tags netgo -mod=vendor -ldflags="-X 'main.Version=%{version}' -X 'main.Builder=OBS' -X 'main.BuildTime=${SOURCE_DATE_EPOCH}'" -o %{name} .

%install
install -Dpm 0755 %{name} %{buildroot}%{_bindir}/%{name}
install -Dpm 0644 conf/sisyphus.service %{buildroot}%{_unitdir}/sisyphus.service
mkdir -p %{buildroot}/etc/sisyphus
install -Dpm 0644 conf/config.yml.example %{buildroot}/etc/sisyphus/config.yml.example

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%files
%{_bindir}/%{name}
%{_unitdir}/sisyphus.service
%{_sysconfdir}/sisyphus/config.yml.example
%doc README.md
%license COPYING

%changelog
* Thu Nov 11 2021 John Seekins <jseekins@datto.com> - 0.1.6
- Initial packaged release
