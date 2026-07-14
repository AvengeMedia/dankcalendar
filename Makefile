# Root Makefile for Dank Calendar
# Orchestrates the Go core build and local installation of the
# binary (with the quickshell UI embedded), icon, desktop entry,
# and systemd unit.

BINARY_NAME=dcal
# Quickshell config dir name (kept as dankcal for config-path compatibility).
SHELL_NAME=dankcal
# Icon / app name (matches the desktop AppId com.danklinux.dankcalendar).
ICON_NAME=dankcalendar
CORE_DIR=core
BUILD_DIR=$(CORE_DIR)/bin
PREFIX ?= /usr/local
DESTDIR ?=
INSTALL_DIR=$(PREFIX)/bin
DATA_DIR=$(PREFIX)/share
ICON_DIR=$(DATA_DIR)/icons/hicolor/scalable/apps
APPLICATIONS_DIR=$(DATA_DIR)/applications

USER_HOME := $(if $(SUDO_USER),$(shell getent passwd $(SUDO_USER) | cut -d: -f6),$(HOME))
# Honor XDG_CONFIG_HOME for the user systemd unit (NixOS and custom setups);
# fall back to ~/.config. Under sudo we can't read the target user's env, so
# use their home's ~/.config.
USER_CONFIG_HOME := $(if $(SUDO_USER),$(USER_HOME)/.config,$(or $(XDG_CONFIG_HOME),$(USER_HOME)/.config))
SYSTEMD_USER_DIR=$(USER_CONFIG_HOME)/systemd/user

SHELL_DIR=quickshell
# Legacy install location, kept for uninstall cleanup only; the UI is
# embedded in the binary since the single-binary distribution change.
SHELL_INSTALL_DIR=$(DATA_DIR)/quickshell/$(SHELL_NAME)
ASSETS_DIR=assets
DESKTOP_ID=com.danklinux.dankcalendar

.PHONY: all build dev run clean test fmt vet migrate migrate-checksum update-common i18n-extract i18n-local i18n-test i18n-push i18n-sync i18n-check install install-bin install-icon install-desktop install-systemd uninstall uninstall-bin uninstall-shell uninstall-icon uninstall-desktop uninstall-systemd help

all: build

build:
	@$(MAKE) -C $(CORE_DIR) build

dev:
	@$(MAKE) -C $(CORE_DIR) dev

run: dev
	@$(BUILD_DIR)/$(BINARY_NAME) run -c $(CURDIR)/$(SHELL_DIR)

clean:
	@$(MAKE) -C $(CORE_DIR) clean

test:
	@$(MAKE) -C $(CORE_DIR) test

fmt:
	@$(MAKE) -C $(CORE_DIR) fmt

vet:
	@$(MAKE) -C $(CORE_DIR) vet

migrate:
	@$(MAKE) -C $(CORE_DIR) migrate name=$(name)

migrate-checksum:
	@$(MAKE) -C $(CORE_DIR) migrate-checksum

# Pull the latest dank-qml-common and pin it everywhere it is consumed
# (submodule pointer + nix flake input). Commit both in one change.
update-common:
	git submodule update --remote --merge dank-qml-common
	nix flake update dank-qml-common

i18n-extract:
	@python3 $(SHELL_DIR)/translations/extract_translations.py

i18n-local:
	@python3 $(SHELL_DIR)/scripts/i18nsync.py local

i18n-test:
	@python3 $(SHELL_DIR)/scripts/i18nsync.py test

i18n-push:
	@python3 $(SHELL_DIR)/scripts/i18nsync.py push

i18n-sync:
	@python3 $(SHELL_DIR)/scripts/i18nsync.py sync

i18n-check:
	@python3 $(SHELL_DIR)/scripts/i18nsync.py check

install-bin:
	@test -f $(BUILD_DIR)/$(BINARY_NAME) || { echo "$(BUILD_DIR)/$(BINARY_NAME) not found; run 'make' first"; exit 1; }
	@echo "Installing $(BINARY_NAME) to $(DESTDIR)$(INSTALL_DIR)..."
	@install -D -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(DESTDIR)$(INSTALL_DIR)/$(BINARY_NAME)

install-icon:
	@echo "Installing icon..."
	@install -D -m 644 $(ASSETS_DIR)/$(ICON_NAME).svg $(DESTDIR)$(ICON_DIR)/$(ICON_NAME).svg
	@test -n "$(DESTDIR)" || gtk-update-icon-cache -q $(DATA_DIR)/icons/hicolor 2>/dev/null || true

install-desktop:
	@echo "Installing desktop entry..."
	@install -D -m 644 $(ASSETS_DIR)/$(DESKTOP_ID).desktop $(DESTDIR)$(APPLICATIONS_DIR)/$(DESKTOP_ID).desktop
	@test -n "$(DESTDIR)" || update-desktop-database -q $(APPLICATIONS_DIR) 2>/dev/null || true

install-systemd:
	@echo "Installing systemd user service to $(SYSTEMD_USER_DIR)..."
	@mkdir -p $(SYSTEMD_USER_DIR)
	@sed 's|/usr/bin/dcal|$(INSTALL_DIR)/$(BINARY_NAME)|g' $(ASSETS_DIR)/systemd/$(BINARY_NAME).service > $(SYSTEMD_USER_DIR)/$(BINARY_NAME).service
	@chmod 644 $(SYSTEMD_USER_DIR)/$(BINARY_NAME).service
	@if [ -n "$(SUDO_USER)" ]; then chown $(SUDO_USER) $(SYSTEMD_USER_DIR)/$(BINARY_NAME).service; fi

install: install-bin install-icon install-desktop
	@echo ""
	@echo "Installation complete."
	@echo "Launch with 'dcal show' or the Dank Calendar desktop entry."
	@echo "Optional: 'make install-systemd' then 'systemctl --user enable --now dcal'."

uninstall-bin:
	@rm -f $(DESTDIR)$(INSTALL_DIR)/$(BINARY_NAME)

uninstall-shell:
	@rm -rf $(DESTDIR)$(SHELL_INSTALL_DIR)

uninstall-icon:
	@rm -f $(DESTDIR)$(ICON_DIR)/$(ICON_NAME).svg
	@test -n "$(DESTDIR)" || gtk-update-icon-cache -q $(DATA_DIR)/icons/hicolor 2>/dev/null || true

uninstall-desktop:
	@rm -f $(DESTDIR)$(APPLICATIONS_DIR)/$(DESKTOP_ID).desktop
	@test -n "$(DESTDIR)" || update-desktop-database -q $(APPLICATIONS_DIR) 2>/dev/null || true

uninstall-systemd:
	@rm -f $(SYSTEMD_USER_DIR)/$(BINARY_NAME).service
	@echo "Stop/disable the service manually if running: systemctl --user disable --now $(BINARY_NAME)"

uninstall: uninstall-desktop uninstall-icon uninstall-shell uninstall-bin uninstall-systemd
	@echo "Uninstallation complete."

help:
	@echo "Build:"
	@echo "  build              - Build the dcal binary (release flags)"
	@echo "  dev                - Fast development build"
	@echo "  run                - Build and run against the in-repo quickshell config"
	@echo "  clean / test / fmt / vet"
	@echo "  update-common      - Bump the dank-qml-common submodule + flake input"
	@echo "  i18n-extract       - Regenerate translations/en.json from I18n.tr() calls"
	@echo "  i18n-local         - Re-extract and show added/removed terms (no POEditor)"
	@echo "  i18n-test          - Extract and validate, no POEditor calls"
	@echo "  i18n-push          - Force-upload all source terms (use for first upload; needs DCAL_POEDITOR_* env)"
	@echo "  i18n-sync          - Upload changed source terms + download translations (needs DCAL_POEDITOR_* env)"
	@echo "  i18n-check         - Fail if local i18n is out of sync with POEditor"
	@echo ""
	@echo "Install (PREFIX=$(PREFIX)):"
	@echo "  install            - Binary (UI embedded), icon, desktop entry"
	@echo "  install-systemd    - Optional systemd user unit (autostarts with the session)"
	@echo "  uninstall          - Remove everything"
