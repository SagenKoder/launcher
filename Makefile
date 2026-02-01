APP := launcher
VERSION ?= 0.1.3
ARCH ?= $(shell go env GOARCH)
GOOS ?= $(shell go env GOOS)
MAINTAINER ?= Launcher Developers <ops@example.com>
DESCRIPTION ?= Native Go-based launcher with plugin support.
DIST := dist
BUILD := build
GOCACHE := $(CURDIR)/.gocache
BINARY := $(DIST)/$(APP)
DEB_STAGING := $(BUILD)/deb/$(APP)
DEB_CONTROL := $(DEB_STAGING)/DEBIAN
DEB_OUTPUT := $(DIST)/$(APP)_$(VERSION)_$(ARCH).deb

.PHONY: build build-release clean package install-user install-macos bundle-macos install-launchagent uninstall-launchagent

$(DIST):
	mkdir -p $(DIST)

build: | $(DIST)
	GOCACHE=$(GOCACHE) go build -o $(BINARY) ./cmd/launcher

build-release: | $(DIST)
	GOCACHE=$(GOCACHE) GOOS=$(GOOS) GOARCH=$(ARCH) go build -trimpath -ldflags '-s -w -buildid=' -o $(BINARY) ./cmd/launcher

package: build-release
	@[ "$(GOOS)" = "linux" ] || { echo "package target currently only supports Linux (.deb)" >&2; exit 1; }
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

install-macos: build-release
	@[ "$(GOOS)" = "darwin" ] || { echo "install-macos target only supports macOS" >&2; exit 1; }
	install -d "$(HOME)/bin"
	install -m 0755 $(BINARY) "$(HOME)/bin/$(APP)"
	install -d "$(HOME)/Library/Application Support/Launcher"
	@if [ ! -f "$(HOME)/Library/Application Support/Launcher/config.yaml" ]; then \
		install -m 0600 config.example.yaml "$(HOME)/Library/Application Support/Launcher/config.yaml"; \
		echo "Installed config to ~/Library/Application Support/Launcher/config.yaml"; \
	else \
		install -m 0600 config.example.yaml "$(HOME)/Library/Application Support/Launcher/config.yaml.example"; \
		echo "Existing config preserved; wrote example to config.yaml.example"; \
	fi
	@echo "Installed $(APP) to ~/bin/$(APP)"
	@echo "Make sure ~/bin is in your PATH"

BUNDLE := $(DIST)/Launcher.app
BUNDLE_CONTENTS := $(BUNDLE)/Contents
BUNDLE_MACOS := $(BUNDLE_CONTENTS)/MacOS
BUNDLE_RESOURCES := $(BUNDLE_CONTENTS)/Resources

bundle-macos: build-release
	@[ "$(GOOS)" = "darwin" ] || { echo "bundle-macos target only supports macOS" >&2; exit 1; }
	rm -rf $(BUNDLE)
	mkdir -p $(BUNDLE_MACOS) $(BUNDLE_RESOURCES)
	install -m 0755 $(BINARY) $(BUNDLE_MACOS)/$(APP)
	@echo '<?xml version="1.0" encoding="UTF-8"?>' > $(BUNDLE_CONTENTS)/Info.plist
	@echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '<plist version="1.0">' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '<dict>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '  <key>CFBundleExecutable</key><string>$(APP)</string>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '  <key>CFBundleIdentifier</key><string>com.sagenkoder.launcher</string>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '  <key>CFBundleName</key><string>Launcher</string>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '  <key>CFBundleDisplayName</key><string>Launcher</string>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '  <key>CFBundleVersion</key><string>$(VERSION)</string>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '  <key>CFBundleShortVersionString</key><string>$(VERSION)</string>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '  <key>CFBundlePackageType</key><string>APPL</string>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '  <key>LSMinimumSystemVersion</key><string>10.15</string>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '  <key>NSHighResolutionCapable</key><true/>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '</dict>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo '</plist>' >> $(BUNDLE_CONTENTS)/Info.plist
	@echo "Built $(BUNDLE)"
	@echo "To install: cp -r $(BUNDLE) /Applications/"

LAUNCHAGENT_PLIST := $(HOME)/Library/LaunchAgents/com.sagenkoder.launcher.plist
LAUNCHAGENT_LABEL := com.sagenkoder.launcher

install-launchagent: install-macos
	@[ "$(GOOS)" = "darwin" ] || { echo "install-launchagent only supports macOS" >&2; exit 1; }
	@mkdir -p "$(HOME)/Library/LaunchAgents"
	@echo '<?xml version="1.0" encoding="UTF-8"?>' > $(LAUNCHAGENT_PLIST)
	@echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> $(LAUNCHAGENT_PLIST)
	@echo '<plist version="1.0">' >> $(LAUNCHAGENT_PLIST)
	@echo '<dict>' >> $(LAUNCHAGENT_PLIST)
	@echo '  <key>Label</key><string>$(LAUNCHAGENT_LABEL)</string>' >> $(LAUNCHAGENT_PLIST)
	@echo '  <key>ProgramArguments</key>' >> $(LAUNCHAGENT_PLIST)
	@echo '  <array>' >> $(LAUNCHAGENT_PLIST)
	@echo '    <string>$(HOME)/bin/$(APP)</string>' >> $(LAUNCHAGENT_PLIST)
	@echo '    <string>--hidden</string>' >> $(LAUNCHAGENT_PLIST)
	@echo '  </array>' >> $(LAUNCHAGENT_PLIST)
	@echo '  <key>RunAtLoad</key><true/>' >> $(LAUNCHAGENT_PLIST)
	@echo '  <key>KeepAlive</key><false/>' >> $(LAUNCHAGENT_PLIST)
	@echo '</dict>' >> $(LAUNCHAGENT_PLIST)
	@echo '</plist>' >> $(LAUNCHAGENT_PLIST)
	launchctl load $(LAUNCHAGENT_PLIST) 2>/dev/null || true
	@echo "Installed launch agent: $(LAUNCHAGENT_PLIST)"
	@echo "Launcher will auto-start (hidden) on login"

uninstall-launchagent:
	@[ "$(GOOS)" = "darwin" ] || { echo "uninstall-launchagent only supports macOS" >&2; exit 1; }
	launchctl unload $(LAUNCHAGENT_PLIST) 2>/dev/null || true
	rm -f $(LAUNCHAGENT_PLIST)
	@echo "Removed launch agent"
