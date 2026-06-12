# Root Makefile for Dank Calendar
# Orchestrates the Go core build and local installation of the
# binary, quickshell config, icon, desktop entry, and systemd unit.

BINARY_NAME=dankcal
CORE_DIR=core
BUILD_DIR=$(CORE_DIR)/bin
PREFIX ?= /usr/local
INSTALL_DIR=$(PREFIX)/bin
DATA_DIR=$(PREFIX)/share
ICON_DIR=$(DATA_DIR)/icons/hicolor/scalable/apps
APPLICATIONS_DIR=$(DATA_DIR)/applications

USER_HOME := $(if $(SUDO_USER),$(shell getent passwd $(SUDO_USER) | cut -d: -f6),$(HOME))
SYSTEMD_USER_DIR=$(USER_HOME)/.config/systemd/user

SHELL_DIR=quickshell
SHELL_INSTALL_DIR=$(DATA_DIR)/quickshell/$(BINARY_NAME)
ASSETS_DIR=assets
DESKTOP_ID=com.danklinux.dankcal

.PHONY: all build dev run clean test fmt vet i18n-extract install install-bin install-shell install-icon install-desktop install-systemd uninstall uninstall-bin uninstall-shell uninstall-icon uninstall-desktop uninstall-systemd help

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

i18n-extract:
	@python3 $(SHELL_DIR)/translations/extract_translations.py

install-bin:
	@test -f $(BUILD_DIR)/$(BINARY_NAME) || { echo "$(BUILD_DIR)/$(BINARY_NAME) not found; run 'make' first"; exit 1; }
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@install -D -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

install-shell:
	@echo "Installing shell files to $(SHELL_INSTALL_DIR)..."
	@mkdir -p $(SHELL_INSTALL_DIR)
	@cp -r $(SHELL_DIR)/* $(SHELL_INSTALL_DIR)/

install-icon:
	@echo "Installing icon..."
	@install -D -m 644 $(SHELL_DIR)/assets/$(BINARY_NAME).svg $(ICON_DIR)/$(BINARY_NAME).svg
	@gtk-update-icon-cache -q $(DATA_DIR)/icons/hicolor 2>/dev/null || true

install-desktop:
	@echo "Installing desktop entry..."
	@install -D -m 644 $(ASSETS_DIR)/$(DESKTOP_ID).desktop $(APPLICATIONS_DIR)/$(DESKTOP_ID).desktop
	@update-desktop-database -q $(APPLICATIONS_DIR) 2>/dev/null || true

install-systemd:
	@echo "Installing systemd user service to $(SYSTEMD_USER_DIR)..."
	@mkdir -p $(SYSTEMD_USER_DIR)
	@sed 's|/usr/bin/dankcal|$(INSTALL_DIR)/$(BINARY_NAME)|g' $(ASSETS_DIR)/systemd/$(BINARY_NAME).service > $(SYSTEMD_USER_DIR)/$(BINARY_NAME).service
	@chmod 644 $(SYSTEMD_USER_DIR)/$(BINARY_NAME).service
	@if [ -n "$(SUDO_USER)" ]; then chown $(SUDO_USER) $(SYSTEMD_USER_DIR)/$(BINARY_NAME).service; fi

install: install-bin install-shell install-icon install-desktop
	@echo ""
	@echo "Installation complete."
	@echo "Launch with 'dankcal show' or the Dank Calendar desktop entry."
	@echo "Optional: 'make install-systemd' then 'systemctl --user enable --now dankcal'."

uninstall-bin:
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)

uninstall-shell:
	@rm -rf $(SHELL_INSTALL_DIR)

uninstall-icon:
	@rm -f $(ICON_DIR)/$(BINARY_NAME).svg
	@gtk-update-icon-cache -q $(DATA_DIR)/icons/hicolor 2>/dev/null || true

uninstall-desktop:
	@rm -f $(APPLICATIONS_DIR)/$(DESKTOP_ID).desktop
	@update-desktop-database -q $(APPLICATIONS_DIR) 2>/dev/null || true

uninstall-systemd:
	@rm -f $(SYSTEMD_USER_DIR)/$(BINARY_NAME).service
	@echo "Stop/disable the service manually if running: systemctl --user disable --now $(BINARY_NAME)"

uninstall: uninstall-desktop uninstall-icon uninstall-shell uninstall-bin uninstall-systemd
	@echo "Uninstallation complete."

help:
	@echo "Build:"
	@echo "  build              - Build the dankcal binary (release flags)"
	@echo "  dev                - Fast development build"
	@echo "  run                - Build and run against the in-repo quickshell config"
	@echo "  clean / test / fmt / vet"
	@echo "  i18n-extract       - Regenerate translations/en.json from I18n.tr() calls"
	@echo ""
	@echo "Install (PREFIX=$(PREFIX)):"
	@echo "  install            - Binary, shell files, icon, desktop entry"
	@echo "  install-systemd    - Optional systemd user unit (autostarts with the session)"
	@echo "  uninstall          - Remove everything"
