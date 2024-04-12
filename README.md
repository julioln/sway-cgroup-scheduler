# sway-cgroup-scheduler

Sets the priority of application cgroups with Sway integration, for a better desktop experience.

The idea is that applications get more CPU priority over background processes, and applications with visible windows get more CPU priority over applications without visible windows.

The scheduler relies on a cgroup hierarchy that is organized in slices within the user's session (defualt systemd behavior). For even better results, place the sway process in its own slice `sway.slice` that will get the highest priority possible, ensuring sway itself doesn't drop inputs or frames, along with other control processes like `waybar` or `swayidle`, even when the host is heavily congested.

Applications should be spawned within `app.slice`.

You can check your cgroup setup with `systemd-cgls`. If you want to spawn a process within a specific slice, you can use `systemd-run --user --scope --slice=app.slice <command>`. In Sway's config, you can use this to load a service into the `session.slice` which runs background processes: `exec systemd-run --user --slice=session.slice --unit=nm-applet-icon /usr/bin/nm-applet --indicator`. To load the scheduler itself, for example, you can use `exec systemd-run --user --slice=sway.slice --unit=sway-cgroup-scheduler /usr/local/bin/sway-cgroup-scheduler`.

## How does it work

The scheduler will connect to the Sway IPC and listen for events. Everytime a window is created, closed, focused, has modes changed, workspaces get focused or created – anything that can cause window visibility to change – the scheduler will get a list of visible pids and find their respective cgroups within the user's `app.slice`. A CPU weight of `1000` (equivalent to niceness -10) will be given to visible applications, non-visible will be kept at the original weight or `100` (niceness 0), any other cgroups will be given CPu weight of `10` (niceness 10).

This should ensure that visible windows have a better chance at presenting frames, processing audio or showing whatever the user's doing that is visible. You can set a heavy compilation job, and unless you are looking at it, you shouldn't notice it's consuming all your CPU time. The UI of visible windows should remain responsive.

The scheduler itself should be very lightweight and run fast. No perceived latency should be felt by using this scheduler, especially if placed in `sway.slice`.

## Example of CPU weights with the scheduler running

The cgroups of visible applications will have a weight of `1000`.

```
$ grep -R . /sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/*/cpu.weight

/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/app-dbus\x2d:1.8\x2dorg.a11y.atspi.Registry.slice/cpu.weight:10
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dbus.socket/cpu.weight:10
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dconf.service/cpu.weight:10
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dunst.service/cpu.weight:10
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/gcr-ssh-agent.socket/cpu.weight:10
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-0df316a1a9428ad8252642b52f80d12cfffd4995bf4ecdcf17daf8e2f740532f.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-1bbf8cb567b144e0aa8a8a4801530884d306c534a5b22113458cb7e25e160a96.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-4d076c8a5604bec978226f311eb1cb530affbbeb6c4642c92052c67e7d8a4e97.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-aaef58d2e9f680989177e7198088deb37994a662097c48d0892596ede300933e.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-bd66584f9f238e40e84c133d476c418e1fea66129434e254bdbf2f8cd6765d92.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-c010061e7a288a75051c198299a9b368475f382cc00c489aac5dadfef81ac9ee.scope/cpu.weight:1000
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-ce9693a3b22de527d96bc8654759c0c198da29a13980b7425d7e48adfe8d3320.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-conmon-0df316a1a9428ad8252642b52f80d12cfffd4995bf4ecdcf17daf8e2f740532f.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-conmon-1bbf8cb567b144e0aa8a8a4801530884d306c534a5b22113458cb7e25e160a96.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-conmon-4d076c8a5604bec978226f311eb1cb530affbbeb6c4642c92052c67e7d8a4e97.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-conmon-aaef58d2e9f680989177e7198088deb37994a662097c48d0892596ede300933e.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-conmon-bd66584f9f238e40e84c133d476c418e1fea66129434e254bdbf2f8cd6765d92.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-conmon-c010061e7a288a75051c198299a9b368475f382cc00c489aac5dadfef81ac9ee.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-conmon-ce9693a3b22de527d96bc8654759c0c198da29a13980b7425d7e48adfe8d3320.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-conmon-f1569da3aa98f6c146283ecfac12924af06e3136e3b31c1c1fee20433ef5ee21.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-f1569da3aa98f6c146283ecfac12924af06e3136e3b31c1c1fee20433ef5ee21.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/run-r3359edc948d4473097a70722204a539a.scope/cpu.weight:1000
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/run-r5aa0160f4fc848bbaa72ff894e15ed8e.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/run-r7c46d3051b3642ebbac850e1a835c366.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/run-r83035532c71e4a8b9a1bf0714a70fba7.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/run-r9dc66755c5aa440593a7d9fb04d74492.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/run-r9f76724ae4314ff890a5654a2f3a0447.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/run-rcc5e5ac1613946fe9f22759eeea0c113.scope/cpu.weight:100 
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/run-rd631734907ac448f989cdd581087faac.scope/cpu.weight:100
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/xdg-desktop-portal-gtk.service/cpu.weight:10
/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/xdg-desktop-portal-wlr.service/cpu.weight:10
```

# Building

A Makefile is provided. To build and install, run `make all`. The binary will be in whatever default golang directory holds your compiled binaries.
