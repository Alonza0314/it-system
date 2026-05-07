# Runner deb

## Files

- Runner Binary: `/usr/bin/it-system-runner`
- Configuration: `/etc/it-system/config-runner.yaml`
- System Service: `/lib/systemd/system/it-system-runner.service`
- Runtime Data: `/usr/share/it-system-runner/.db`
- Token File (default): `/usr/share/it-system-runner/.db/token`

## Release Script

```bash
./deb/release_runner.sh <version>
```

Examples:

```bash
./deb/release_runner.sh v1.2.3
./deb/release_runner.sh 1.2.3
```

This generates `pkg/it-system-runner_<version>_amd64.deb`.
If input is `v1.2.3`, output version becomes `1.2.3`.

## Install deb

```bash
sudo dpkg -i pkg/it-system-runner_<version>_amd64.deb
```

Runtime dependencies are required on runner host because task execution runs `git clone` and `make` for free5GC:

- `git`
- `make`
- `go` (any installation source is fine, as long as `go` is in PATH)

Quick check:

```bash
which git make go
```

During install when runner token is missing (for example fresh install or reinstall), installer asks for:

- runner name
- runner ip
- controller ip
- controller port
- controller admin username
- controller admin password

Installer flow:

1. Update `/etc/it-system/config-runner.yaml` fields: `name`, `controller_ip`, `controller_port`
2. Login controller via `POST /api/login`
3. Register runner via `POST /api/admin/runner` with payload:

    ```json
    {
    "name": "runner",
    "ip": "10.0.0.1"
    }
    ```

4. Read response token and write it to `token_path` in config (default: `.db/token` => `/usr/share/it-system-runner/.db/token`)
5. Enable and start `it-system-runner` service automatically

If registration fails, installation stops.

## Upgrade

```bash
sudo dpkg -i pkg/it-system-runner_<new-version>_amd64.deb
```

## Remove

```bash
sudo dpkg -r it-system-runner
```

During remove, installer:

1. stops and disables `it-system-runner` service
2. removes all files under `/usr/share/it-system-runner/.db`

Remove does not delete config files.

If you want to remove config too:

```bash
sudo dpkg -P it-system-controller
```

If you also want to remove storage files:

```bash
sudo rm -rf /usr/share/it-system-runner
```

## Journal Logs

```bash
# recent 100 lines
sudo journalctl -u it-system-runner -n 100 --no-pager

# follow in real time
sudo journalctl -u it-system-runner -f

# since current boot
sudo journalctl -u it-system-runner -b --no-pager
```
