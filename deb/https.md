# HTTPS

The controller serves plain HTTP on port 8888, so passwords and login tokens
cross the network unencrypted. To serve it over HTTPS, put a reverse proxy in
front of it. This guide uses [Caddy](https://caddyserver.com), which gets and
renews a Let's Encrypt certificate by itself, redirects HTTP to HTTPS, and
passes WebSocket connections through without extra settings.

```text
browser --HTTPS :443--> Caddy --HTTP--> controller 127.0.0.1:8888
runner  --HTTPS :443--> Caddy             (controller_url)
runner  --HTTP  :8888 over a private network (VPN) --> controller
```

## Requirements

- A DNS name for the controller host, such as `it.example.org`.
- Ports 80 and 443 reachable from the internet (Let's Encrypt checks the name
  over them).

## 1. Install Caddy

Follow the Debian/Ubuntu instructions at
<https://caddyserver.com/docs/install#debian-ubuntu-raspbian>; they install
Caddy as the `caddy` service.

## 2. Proxy to the controller

`/etc/caddy/Caddyfile`:

```caddyfile
it.example.org {
	reverse_proxy 127.0.0.1:8888
}
```

```bash
sudo systemctl reload caddy
```

## 3. Trust the proxy for client addresses

In `/etc/it-system/config-controller.yaml`, under `backend`:

```yaml
  trustedProxies: ["127.0.0.1"]
```

Without it the controller logs every request as coming from `127.0.0.1`.
Only list proxies you run: a trusted proxy can claim any client address.

```bash
sudo systemctl restart it-system-controller
```

## 4. Close port 8888 to the public

Otherwise anyone can still log in over plain HTTP on port 8888. Keep 8888
open only to the host itself and to the network your runners use. For example
with `ufw`, if runners reach the controller over `10.8.0.0/16`:

```bash
sudo ufw allow OpenSSH          # before enabling ufw on a remote host
sudo ufw allow 80,443/tcp
sudo ufw allow from 10.8.0.0/16 to any port 8888 proto tcp
sudo ufw deny 8888/tcp
sudo ufw enable
```

## 5. Runners

- A runner that reaches the controller over a private network (VPN) can keep
  `controller_ip` and `controller_port`.
- Any other runner should go through HTTPS. In
  `/etc/it-system/config-runner.yaml`, under `runner`:

  ```yaml
    controller_url: "https://it.example.org"
  ```

  `controller_url` replaces `controller_ip` and `controller_port`. Then
  `sudo systemctl restart it-system-runner`. A new runner install asks for
  this URL when it registers.

## 6. Check

```bash
curl -sI https://it.example.org | head -n 1      # HTTP/2 200
curl -sI http://it.example.org | grep -i location # redirect to https
```

Open `https://it.example.org`, log in, and check that the Runner page lists
the runners as idle. From outside the runner network,
`curl -m 5 http://<public IP>:8888` should time out.
