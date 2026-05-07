# Controller deb

## Files

- Controller Binary: `/usr/bin/it-system-controller`
- Configuration: `/etc/it-system/config-controller.yaml`
- System Service: `/lib/systemd/system/it-system-controller.service`
- Frontend Static File: `/usr/share/it-system-controller/frontend`
- DB: `/usr/share/it-system-controller/.db/it-system.db`
- Log: `/usr/share/it-system-controller/.db/log`

## Release Script

```bash
./deb/release_controller.sh <version>
```

Examples:

```bash
./deb/release_controller.sh v1.2.3
./deb/release_controller.sh 1.2.3
```

This generates `pkg/it-system-controller_<version>_amd64.deb`.
If input is `v1.2.3`, output version becomes `1.2.3`.

## Install deb

```bash
sudo dpkg -i pkg/it-system-controller_<version>_amd64.deb
```

During install, installer asks for admin password:

- press Enter: use default `0000`
- input value: update admin password in config

Installer flow:

1. Update `/etc/it-system/config-controller.yaml` field `password`
2. Enable and start `it-system-controller` service automatically

If result in dependency issue:

```bash
sudo apt-get -f install

sudo systemctl daemon-reload
sudo systemctl enable --now it-system-controller
sudo systemctl status it-system-controller
```

## Upgrade

```bash
sudo dpkg -i pkg/it-system-controller_<new-version>_amd64.deb
```

## Remove

```bash
sudo dpkg -r it-system-controller
```

During remove, installer will stop and disable `it-system-controller` service.

Remove does not delete config and DB/log files.

If you want to remove config too:

```bash
sudo dpkg -P it-system-controller
```

If you also want to remove runtime DB/log files:

```bash
sudo rm -rf /usr/share/it-system-controller
```

## Journal Logs

```bash
# recent 100 lines
sudo journalctl -u it-system-controller -n 100 --no-pager

# follow in real time
sudo journalctl -u it-system-controller -f

# since current boot
sudo journalctl -u it-system-controller -b --no-pager
```
