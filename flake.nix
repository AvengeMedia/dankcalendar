{
  description = "DankCalendar";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs, ... }:
    let
      goModVersion =
        let
          content = builtins.readFile ./core/go.mod;
          lines = builtins.filter builtins.isString (builtins.split "\n" content);
          goLines = builtins.filter (l: builtins.match "go [0-9]+\\..*" l != null) lines;
          matched =
            if goLines != [ ] then builtins.match "go ([0-9]+)\\.([0-9]+).*" (builtins.head goLines) else null;
        in
        if matched != null then
          {
            major = builtins.elemAt matched 0;
            minor = builtins.elemAt matched 1;
          }
        else
          {
            major = "1";
            minor = "25";
          };
      goForPkgs = pkgs: pkgs.${"go_${goModVersion.major}_${goModVersion.minor}"};

      forEachSystem =
        fn:
        nixpkgs.lib.genAttrs [ "aarch64-linux" "x86_64-linux" ] (
          system: fn system nixpkgs.legacyPackages.${system}
        );

      mkModuleWithDcalPkgs =
        modulePath:
        args@{ pkgs, ... }:
        {
          imports = [
            (import modulePath (args // { dcalPkgs = buildDcalPkgs pkgs; }))
          ];
        };

      mkDcal =
        pkgs:
        (
          let
            version = "0.2.0";
          in
          (pkgs.buildGoModule.override { go = goForPkgs pkgs; }) {
            inherit version;
            pname = "dankcalendar";
            src = ./core;
            vendorHash = "sha256-2eBwE1jnvGDQiMD1wKDTIr2CnKWWdhNpIHhkl2R2jIQ=";

            subPackages = [ "cmd/dcal" ];

            ldflags = [
              "-s"
              "-w"
              "-X main.Version=${version}"
              "-X main.Commit=${self.shortRev or self.dirtyShortRev or "dirty"}"
            ];

            nativeBuildInputs = with pkgs; [
              installShellFiles
              makeWrapper
            ];

            postInstall = ''
              mkdir -p $out/share/quickshell/dankcalendar
              cp -r ${./quickshell}/. $out/share/quickshell/dankcalendar/

              install -Dm644 ${./assets/com.danklinux.dankcalendar.desktop} \
                $out/share/applications/com.danklinux.dankcalendar.desktop
              install -Dm644 ${./quickshell/assets/dankcalendar.svg} \
                $out/share/icons/hicolor/scalable/apps/dankcalendar.svg

              install -Dm644 ${./assets/systemd/dcal.service} \
                $out/lib/systemd/user/dcal.service
              substituteInPlace $out/lib/systemd/user/dcal.service \
                --replace-fail /usr/bin/dcal $out/bin/dcal

              wrapProgram $out/bin/dcal \
                --add-flags "-c $out/share/quickshell/dankcalendar"

              installShellCompletion --cmd dcal \
                --bash <($out/bin/dcal completion bash) \
                --fish <($out/bin/dcal completion fish) \
                --zsh <($out/bin/dcal completion zsh)
            '';

            meta = {
              description = "Local, Google, Microsoft, and CalDAV calendars for the dank desktop";
              homepage = "https://github.com/AvengeMedia/dankcalendar";
              license = pkgs.lib.licenses.mit;
              mainProgram = "dcal";
              platforms = pkgs.lib.platforms.linux;
            };
          }
        );

      buildDcalPkgs = pkgs: {
        dankcalendar = mkDcal pkgs;
      };
    in
    {
      packages = forEachSystem (
        system: pkgs: {
          dankcalendar = mkDcal pkgs;
          default = self.packages.${system}.dankcalendar;
        }
      );

      lib = { inherit mkDcal buildDcalPkgs; };

      homeModules.dank-calendar = mkModuleWithDcalPkgs ./distro/nix/home.nix;

      homeModules.default = self.homeModules.dank-calendar;

      nixosModules.dank-calendar = mkModuleWithDcalPkgs ./distro/nix/nixos.nix;

      nixosModules.default = self.nixosModules.dank-calendar;

      devShells = forEachSystem (
        system: pkgs: {
          default = pkgs.mkShell {
            # Let the golangci-lint pre-commit hook fetch the Go version core/go.mod
            # requires rather than pinning it (prek forces GOTOOLCHAIN=local otherwise).
            GOTOOLCHAIN = "auto";

            buildInputs = with pkgs; [
              (goForPkgs pkgs)
              go-mockery
              gopls
              delve
              go-tools
              gnumake

              quickshell

              prek
              uv
              python3

              nixd
              nil
            ];

            shellHook = ''
              touch quickshell/.qmlls.ini 2>/dev/null
              if [ ! -f .git/hooks/pre-commit ]; then prek install; fi
            '';
          };
        }
      );

      nixosTests = nixpkgs.lib.genAttrs [ "aarch64-linux" "x86_64-linux" ] (
        system:
        import ./distro/nix/tests {
          inherit self;
          pkgs = nixpkgs.legacyPackages.${system};
        }
      );
    };
}
