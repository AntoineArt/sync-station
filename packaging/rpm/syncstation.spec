Name:           syncstation
Version:        1.0.0
Release:        1%{?dist}
Summary:        CLI tool for syncing configuration files across computers

License:        GPL-3.0
URL:            https://github.com/AntoineArt/syncstation
Source0:        https://github.com/AntoineArt/syncstation/archive/v%{version}.tar.gz

BuildRequires:  golang >= 1.18
BuildRequires:  git
Requires:       (none)

%description
Sync Station is a cross-platform CLI tool that synchronizes configuration files
between multiple computers using your existing cloud storage (Dropbox, OneDrive, etc.).

Features include interactive TUI interface, cross-platform support, hash-based 
file tracking, conflict detection, and cloud-agnostic operation.

%prep
%autosetup -n %{name}-%{version}-cli

%build
export CGO_ENABLED=0
export GOFLAGS="-buildmode=pie -trimpath"
go build -ldflags="-s -w -X main.Version=%{version}-cli" -o %{name} .

%install
# Install binary
install -Dm755 %{name} %{buildroot}%{_bindir}/%{name}

# Install documentation
install -Dm644 README.md %{buildroot}%{_docdir}/%{name}/README.md
install -Dm644 CONVERSION_DISCUSSION.md %{buildroot}%{_docdir}/%{name}/CONVERSION_DISCUSSION.md


%files
%license LICENSE
%doc README.md CONVERSION_DISCUSSION.md
%{_bindir}/%{name}

%post
echo "Sync Station installed successfully!"
echo "Run 'syncstation --help' to get started."
echo "Initialize with: syncstation init --cloud-dir /path/to/your/cloud/folder"

%changelog
* Thu Jul 18 2025 Sync Station Team <noreply@example.com> - 1.0.0-1
- Complete rewrite from GUI to CLI/TUI
- Cross-platform config system with XDG compliance
- Interactive TUI with bulk operations
- Hash-based file tracking
- Package manager support