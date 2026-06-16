{
  lib,
  dcalPkgs,
  pkgs,
  ...
}:
let
  inherit (lib) types;
in
{
  options.programs.dank-calendar = {
    enable = lib.mkEnableOption "DankCalendar";

    package = lib.mkPackageOption dcalPkgs "dankcalendar" {
      extraDescription = "The DankCalendar package to use (defaults to be built from source)";
    };

    quickshell = {
      package = lib.mkPackageOption pkgs "quickshell" {
        extraDescription = "The Quickshell package used to launch the calendar UI";
      };
    };

    systemd = {
      enable = lib.mkEnableOption "DankCalendar systemd startup";

      target = lib.mkOption {
        type = types.str;
        default = "graphical-session.target";
        description = "Systemd target that will automatically start dcal.";
      };

      restartIfChanged = lib.mkOption {
        type = types.bool;
        default = true;
        description = "Auto-restart dcal.service when dank-calendar changes";
      };
    };
  };
}
