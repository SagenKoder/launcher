APP := launcher
VERSION ?= 0.1.3
ARCH ?= amd64
GOOS ?= linux
MAINTAINER ?= Launcher Developers <ops@example.com>
DESCRIPTION ?= Native Go-based launcher with plugin support.
DIST := dist
BUILD := build
GOCACHE := $(CURDIR)/.gocache
BINARY := $(DIST)/$(APP)
DEB_STAGING := $(BUILD)/deb/$(APP)
DEB_CONTROL := $(DEB_STAGING)/DEBIAN
DEB_OUTPUT := $(DIST)/$(APP)_$(VERSION)_$(ARCH).deb

.PHONY: build build-release clean package install-user

$(DIST):
	mkdir -p $(DIST)

build: | $(DIST)
	GOCACHE=$(GOCACHE) go build -o $(BINARY) ./cmd/launcher

build-release: | $(DIST)
	GOCACHE=$(GOCACHE) GOOS=$(GOOS) GOARCH=$(ARCH) go build -trimpath -ldflags '-s -w -buildid=' -o $(BINARY) ./cmd/launcher

package: build-release
	rm -rf $(DEB_STAGING)
	mkdir -p $(DEB_CONTROL)
	mkdir -p $(DEB_STAGING)/usr/bin
	mkdir -p $(DEB_STAGING)/etc/launcher
	install -m 0755 $(BINARY) $(DEB_STAGING)/usr/bin/$(APP)
	install -m 0600 config.example.yaml $(DEB_STAGING)/etc/launcher/config.yaml
	printf 'Package: %s\nVersion: %s\nSection: utils\nPriority: optional\nArchitecture: %s\nMaintainer: %s\nDescription: %s\n' $(APP) $(VERSION) $(ARCH) "$(MAINTAINER)" "$(DESCRIPTION)" > $(DEB_CONTROL)/control
	echo '/etc/launcher/config.yaml' > $(DEB_CONTROL)/conffiles
	dpkg-deb --build --root-owner-group $(DEB_STAGING) $(DEB_OUTPUT)
	@echo "Built $(DEB_OUTPUT)"

clean:
	rm -rf $(DIST) $(BUILD)

install-user: build-release
	install -d $(HOME)/.local/bin
	install -m 0755 $(BINARY) $(HOME)/.local/bin/$(APP)
	install -d $(HOME)/.config/launcher
	if [ ! -f $(HOME)/.config/launcher/config.yaml ]; then \
		install -m 0600 config.example.yaml $(HOME)/.config/launcher/config.yaml; \
	else \
		install -m 0600 config.example.yaml $(HOME)/.config/launcher/config.yaml.example; \
		echo "Existing config preserved; wrote example to config.yaml.example"; \
	fi
