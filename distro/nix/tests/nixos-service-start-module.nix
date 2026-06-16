{
  self,
  pkgs,
  ...
}:
pkgs.testers.runNixOSTest {
  name = "dankcalendar-nixos-service-start-module";

  nodes.machine = {
    imports = [
      self.nixosModules.dank-calendar
    ];

    users.users.danklinux = {
      isNormalUser = true;
      extraGroups = [ "wheel" ];
    };

    programs.dank-calendar = {
      enable = true;
      systemd = {
        enable = true;
        target = "default.target";
      };
    };

    system.stateVersion = "25.11";
  };

  testScript = ''
    machine.wait_for_unit("multi-user.target")

    unit = "/run/current-system/sw/share/systemd/user/dcal.service"
    machine.succeed(f"test -f {unit}")
    machine.succeed(f"grep -q 'run --session --hidden' {unit}")
    machine.succeed("test -e /etc/systemd/user/default.target.wants/dcal.service")
  '';
}
