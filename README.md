# Wireguard UI (solutions.io build)

A web interface for managing WireGuard: clients, server settings, and which traffic of VPN clients the server forwards. It runs on our VPN server, localnetwork.solutions.io.

This is our own build of [ngoduykhanh/wireguard-ui](https://github.com/ngoduykhanh/wireguard-ui) v0.6.2 (MIT license). Upstream has not been updated since 2024, so we maintain this build ourselves.

## Differences from upstream v0.6.2

- **Firewall page**: limit what VPN clients can reach through the server. See [Firewall](#firewall).
- **Updated**: Go 1.27, echo 4.16, gorilla/sessions 1.4, x/crypto 0.59, wgctrl 2024-12, validator v10; AdminLTE 3.2, jQuery 3.8, Bootstrap 4.6.2, Select2 4.1.
- **Security fixes**:
  - Client name, email and endpoint could add lines to `wg0.conf`, including `PostUp`, which wg-quick runs as root.
  - Pages use `html/template`, and the client list escapes its fields (stored XSS).
  - Private keys of clients and of the server are no longer written to the log.
  - The default session secret uses `crypto/rand`; the session cookie is `SameSite=Lax`.
  - The About page no longer calls the GitHub API.
- **Removed**: Telegram, sending configs by email, Wake-on-LAN, and the upstream Docker images and release workflows.

## Firewall

The Firewall page (admins only) has two lists:

| List | Meaning |
|---|---|
| Allowed destinations | Addresses or networks that VPN clients may reach on every port, such as our own servers |
| Allowed TCP and UDP ports | Ports that VPN clients may use towards every other address, such as 80 and 443 |

Replies to allowed connections and ping are always allowed. Everything else from VPN clients is rejected. Clients keep their configuration: nobody has to import anything when the lists change.

How it works on the server:

- Changes are saved first and take effect on **Apply Config**, like all other changes.
- Apply Config writes the rules to `wireguard-ui-firewall.nft` next to `wg0.conf` (or to `WGUI_FIREWALL_FILE_PATH`) and loads them with `nft -f` into the table `inet wireguard_ui`. If nft rejects the rules, the previous rules stay active and the UI shows the error.
- The table has its own forward chain that runs just before the default one. It only rejects traffic, so `/etc/nftables.conf` stays as it is.
- Reloading nftables (`systemctl reload nftables`) flushes the whole ruleset, including this table. Wireguard UI checks every 30 seconds and loads the last applied file again. The same happens after a reboot.
- Traffic to the VPN server itself (SSH, this UI) is not affected.

Check it on the server:

```sh
nft list table inet wireguard_ui               # the loaded rules
nft list counter inet wireguard_ui blocked     # packets rejected since the rules were loaded
```

To turn the firewall off, clear **Enable the firewall**, save, and click Apply Config. The table is then removed.

## Build

Requirements: Go 1.27 or newer, Node.js with npm, and bash.

```sh
./build.sh                  # dist/wireguard-ui-linux-arm64 for the VPN server
GOARCH=amd64 ./build.sh     # dist/wireguard-ui-linux-amd64
```

Always build with `build.sh`. It runs `prepare_assets.sh` first, which copies the frontend libraries and `custom/js` into `assets/`, and the binary embeds `assets/`. A plain `go build` uses whatever is in `assets/` at that moment.

`prepare_assets.sh` installs npm packages with `--ignore-scripts`: admin-lte pulls in summernote, whose install script fails and is not needed. Only the files the templates load are copied into `assets/`.

## Test

```sh
go test ./...
```

The test that checks the generated rules with `nft -c` needs nft and the `NET_ADMIN` capability; elsewhere it is skipped. To run every test, use a Linux container:

```sh
docker run --rm --cap-add NET_ADMIN -v "$PWD:/src:ro" -w /src golang:1.27 \
  sh -c 'apt-get update -qq && apt-get install -y -qq nftables >/dev/null && go test -buildvcs=false ./...'
```

On Windows with Smart App Control, binaries you build yourself are blocked from running. Run the tests and the app in a Linux container instead.

## Deploy to the VPN server

The server runs the binary as the systemd service `wireguard-ui`, from `/usr/local/bin/wireguard-ui`, with its database in `/usr/local/share/wireguard-ui/db` and its settings in `/etc/default/wireguard-ui`. When `wg0.conf` changes, `wg-quick-watcher@wg0.path` restarts WireGuard.

1. Build: `./build.sh`.
2. Copy the binary to the server: `scp dist/wireguard-ui-linux-arm64 root@localnetwork.solutions.io:/usr/local/bin/wireguard-ui.new`.
3. On the server, back up the binary and the database:

   ```sh
   cp -a /usr/local/bin/wireguard-ui /usr/local/bin/wireguard-ui.previous
   tar czf /root/wireguard-ui-db-$(date +%F).tar.gz -C /usr/local/share/wireguard-ui db
   ```

4. Replace the binary and restart the UI. This does not interrupt VPN connections.

   ```sh
   mv /usr/local/bin/wireguard-ui.new /usr/local/bin/wireguard-ui
   systemctl restart wireguard-ui
   systemctl status wireguard-ui
   ```

5. Log in, check the clients and the Firewall page. The first start adds `db/server/firewall.json` with the firewall turned off. The Apply Config button may appear once, because the firewall settings are now part of the config.

To roll back, move `wireguard-ui.previous` back and restart the service.

## Configuration

Environment variables, read at start. On the server they are in `/etc/default/wireguard-ui`.

| Variable | Description | Default |
|---|---|---|
| `BIND_ADDRESS` | Address and port of the web interface, or `unix:///path/to/socket` | `0.0.0.0:5000` |
| `BASE_PATH` | Subpath when the UI runs behind a reverse proxy, e.g. `/wireguard` | none |
| `SESSION_SECRET` / `SESSION_SECRET_FILE` | Secret for the session cookies. Set it; otherwise sessions end on every restart | random |
| `SESSION_MAX_DURATION` | Days a remembered session stays valid | `90` |
| `SUBNET_RANGES` | Named address ranges for new clients, e.g. `Office:10.0.1.0/24; Staff:10.0.2.0/24` | none |
| `WGUI_USERNAME` | Username of the first user (database initialisation only) | `admin` |
| `WGUI_PASSWORD` / `WGUI_PASSWORD_FILE` | Password of the first user (database initialisation only) | `admin` |
| `WGUI_PASSWORD_HASH` / `WGUI_PASSWORD_HASH_FILE` | Bcrypt hash instead of a password (database initialisation only) | none |
| `WGUI_ENDPOINT_ADDRESS` | Endpoint that clients connect to, optionally with a port | public IP of the server |
| `WGUI_FAVICON_FILE_PATH` | Favicon file | WireGuard logo |
| `WGUI_DNS` | Default DNS servers, comma-separated | `1.1.1.1` |
| `WGUI_MTU` | Default MTU | `1450` |
| `WGUI_PERSISTENT_KEEPALIVE` | Default persistent keepalive | `15` |
| `WGUI_FIREWALL_MARK` | WireGuard firewall mark | `0xca6c` |
| `WGUI_TABLE` | WireGuard `Table` setting | `auto` |
| `WGUI_CONFIG_FILE_PATH` | Path of the WireGuard config | `/etc/wireguard/wg0.conf` |
| `WGUI_FIREWALL_FILE_PATH` | Where the firewall rules are written | `wireguard-ui-firewall.nft` next to the WireGuard config |
| `WGUI_LOG_LEVEL` | `DEBUG`, `INFO`, `WARN`, `ERROR` or `OFF` | `INFO` |
| `WG_CONF_TEMPLATE` | Custom template for `wg0.conf` | built-in `templates/wg.conf` |
| `WGUI_SERVER_INTERFACE_ADDRESSES` | Server interface addresses (database initialisation only) | `10.252.1.0/24` |
| `WGUI_SERVER_LISTEN_PORT` | Server listen port (database initialisation only) | `51820` |
| `WGUI_SERVER_POST_UP_SCRIPT` / `WGUI_SERVER_POST_DOWN_SCRIPT` | Server PostUp and PostDown (database initialisation only) | none |
| `WGUI_DEFAULT_CLIENT_ALLOWED_IPS` | Default Allowed IPs for new clients, comma-separated | `0.0.0.0/0` |
| `WGUI_DEFAULT_CLIENT_EXTRA_ALLOWED_IPS` | Default Extra Allowed IPs for new clients | none |
| `WGUI_DEFAULT_CLIENT_USE_SERVER_DNS` | New clients use the DNS servers from the global settings | `true` |
| `WGUI_DEFAULT_CLIENT_ENABLE_AFTER_CREATION` | New clients are enabled right away | `true` |
| `DISABLE_LOGIN` | Turn off authentication. Only for testing | `false` |

## License

MIT, see [LICENSE](LICENSE). Original work by Khanh Ngo and the contributors of [wireguard-ui](https://github.com/ngoduykhanh/wireguard-ui).
