{
  self,
  pkgs,
  ...
}:
rec {
  all = pkgs.symlinkJoin {
    name = "dankcalendar-nixos-tests";
    paths = [
      nixos-module
      nixos-service-start-module
      home-manager-module
    ];
  };

  nixos-module = import ./nixos-module.nix { inherit self pkgs; };

  nixos-service-start-module = import ./nixos-service-start-module.nix { inherit self pkgs; };

  home-manager-module = import ./home-manager-module.nix { inherit self pkgs; };
}
