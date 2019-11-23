Name:       pipefire
Version:    0.9
Release:    11
Summary:    Pipeline based file transfer 
License:    GPLv3+

%description
Pipefire is a pipeline based file transfer utility.

%prep
git clone https://github.com/masenocturnal/pipefire.git

%build
go build -o dist/pipefired ./cmd/pipefired.go

%install
mkdir -p %{buildroot}/usr/bin/
install -m 755 dist/pipefired %{buildroot}/usr/bin/

%files
/usr/bin/pipefired

%changelog
# let's skip this for now
