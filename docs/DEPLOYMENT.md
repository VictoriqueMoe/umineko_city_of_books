# Deployment

Everything past the first `docker compose up`: running the prebuilt image, putting the site behind a reverse proxy, and the three optional LiveKit services that add voice chat, live streaming and smooth HLS playback.

## Prebuilt image

```bash
docker network create observability
docker compose -f docker-compose.prod.yml up -d
```

This pulls `ghcr.io/victoriquemoe/umineko_city_of_books:latest` instead of building locally. The compose file declares `observability` as an **external** network, so it has to exist before the stack will start. The app and a `postgres-exporter` sidecar both join it and carry `prometheus-scrape` labels, so a Prometheus living on that network scrapes the app's internal listener on port `9464` at `/metrics` and the exporter on `9187`.

Two things differ from the dev compose beyond the image source:

- The app publishes on **`127.0.0.1:2312`**, not on every interface, so it is only reachable through a reverse proxy on the same box. It also carries a `wget` healthcheck against `/livez`.
- `valkey`, `livekit`, `livekit-ingress` and `livekit-egress` all run `network_mode: host`, so their ports bind straight onto the host and container-name DNS does not resolve inside them. Every address in `livekit.yaml` / `ingress.yaml` / `egress.yaml` has to be `127.0.0.1:<port>` rather than a compose service name, and the coordination valkey binds `127.0.0.1:6380` to stay clear of a `6379` that is usually already taken.

## Reverse Proxy

Run behind Caddy, Nginx, or similar for TLS. The server sets the right cache headers itself (`/static/assets/*` immutable, `/uploads/*` and `/hls/*` segments long-lived, `.m3u8` playlists `no-cache`, HTML `no-store`, API `no-cache`), so the proxy mostly just forwards. Four things it does have to get right:

- **Preserve the `Host` header.** The host-authorisation middleware answers 403 to any request whose host does not match the hostname of the `base_url` site setting, so a proxy that rewrites Host takes the whole site down. Only `/health`, `/livez` and `/api/v1/livekit/webhook` are exempt.
- **Upgrade WebSocket connections on `/api/v1/ws`**, not `/ws`.
- **Set `CF-Connecting-IP`.** The app is configured with `ProxyHeader: "CF-Connecting-IP"` and trusts loopback and private peers, so behind anything other than Cloudflare the proxy has to set that header itself. Without it every request is attributed to the proxy's own address, and the session IP hash, rate limits and last-seen IP all read it.
- **Never proxy the internal listener.** `/metrics` and `/debug/pprof/*` are unauthenticated and live on `INTERNAL_ADDR` (default `:9464`), not on the public app, so the proxy never sees them. Compose does not publish that port, so only containers on the docker networks can reach it.

`/uploads/*` and `/hls/*` are served by the app on the same origin, so they need no extra proxy config. That stays true with S3 file storage enabled: the app streams bucket objects through `/uploads/*` itself, with range requests and caching headers intact, so the bucket can stay private and the proxy and CDN rules do not change. If you enable voice chat, also add a `wss://` route to the `livekit` container (see below).

## Voice Chat (LiveKit)

Voice chat needs the bundled `livekit` service (already in `docker-compose.yml` / `docker-compose.prod.yml`) plus a small amount of host setup. The code ships disabled, so none of this is required unless you want voice. The same LiveKit setup also powers **in-party voice and screen-share watch parties**, and live streaming builds on top of it. Once voice is configured there is no extra server-side work for those (screen capture is a browser feature served over the existing `wss://`).

1. **Create the config file.** The compose file bind-mounts `./livekit.yaml`, which is gitignored. Copy the template first (otherwise Docker creates an empty directory in its place):

   ```bash
   cp livekit.yaml.example livekit.yaml
   ```

2. **Set the server key/secret.** Edit the `keys:` block in `livekit.yaml` and replace the `devkey` / placeholder secret with a real key name and a long random secret.

3. **Enter the matching values in the app.** In **Admin → Settings → Watch Parties, Voice & Streaming**, toggle **Enable Voice Chat** and fill in:
   - **LiveKit URL**, the public `wss://` URL browsers connect to (see step 4)
   - **API Key** / **API Secret**, the **same** key name and secret you put in `livekit.yaml`

   The app reads these from the database (hot-reloaded, no restart), so they are never stored in `.env`. The three fields appear as soon as either voice chat or live streaming is switched on, and a save that turns voice on while any of them is empty is rejected.

4. **Reverse-proxy the signalling.** Point a public `wss://livekit.example.com` at the `livekit` container's port `7880` and use that URL as the LiveKit URL above.

5. **Open the media port on the host firewall.** LiveKit carries all WebRTC audio over a single UDP port `7882` (with TCP `7881` as a fallback). Open `7882/udp` on the host; without it, calls fail to connect. (A single fixed port avoids the Windows reserved-range binding errors and the slow per-port proxying you get when publishing a large range on Docker Desktop.)

6. **Advertise the right IP.** The production half of `livekit.yaml.example` keeps `rtc.use_external_ip: false` and instead pins `rtc.node_ip` to the box's public IPv4, with `rtc.ips.excludes` dropping IPv6 (`::/0`) and the docker bridge range (`172.16.0.0/12`). Under `network_mode: host` LiveKit otherwise sees every host interface and hands remote peers a docker-bridge or broken-IPv6 candidate, which is how calls end up one-way or silent. For local dev leave that block out, keep `use_external_ip: false`, and let the compose `NODE_IP` env var (`127.0.0.1`) stand in.

7. **TURN (recommended).** Roughly 15-25% of users behind strict NATs need a relay. Enable LiveKit's embedded TURN with a public hostname and TLS certs in `livekit.yaml`, or accept reduced connectivity.

The webhook LiveKit posts back to (`/api/v1/livekit/webhook`) is verified by the shared secret and is exempt from host authorisation, so it never has to go through the public proxy. Point `webhook.urls` at wherever the app is reachable from the LiveKit container: `http://umineko-city-of-books:4323/api/v1/livekit/webhook` on the dev compose's bridge network, and `http://127.0.0.1:2312/api/v1/livekit/webhook` in production, where LiveKit is host-networked and the app publishes on loopback.

## Live Streaming (LiveKit Ingress)

The `/live` page lets any member broadcast from OBS 30+ / Streamlabs (over WHIP) to a public viewer directory anyone can watch without an account. It builds on the Voice Chat setup above and adds the bundled `livekit-ingress` service plus a Valkey (Redis-compatible) coordination bus (both already in the compose files). Disabled by default.

1. **Create the ingress config.** The compose bind-mounts `./ingress.yaml`, which is gitignored. Copy the template (otherwise Docker creates an empty directory in its place):

   ```bash
   cp ingress.yaml.example ingress.yaml
   ```

   Set `api_key` / `api_secret` to the **same** key + secret as `livekit.yaml`.

2. **Wire up the coordination bus.** The ingress and the SFU coordinate over Valkey, so `livekit.yaml` needs a `redis:` block pointing at the **same** instance as `ingress.yaml`. The template does not ship one, add it. That instance is the `valkey` service, **not** `valkey-cache`: the latter is the app's own cache and stays separate. In production both the SFU and the ingress run `network_mode: host`, so they must use `127.0.0.1:<port>` (a host-networked container cannot resolve the `valkey` service name), and `docker-compose.prod.yml` already binds that valkey on `127.0.0.1:6380` to stay clear of a `6379` that is usually taken.

3. **Set the public WHIP endpoint.** Add an `ingress` block to `livekit.yaml` with the public URL OBS/Streamlabs will point at, and reverse-proxy that host to the ingress `whip_port`:

   ```yaml
   # livekit.yaml
   ingress:
     whip_base_url: "https://ingress.example.com/w"
   ```

   ```caddy
   ingress.example.com {
       reverse_proxy :8090   # whatever whip_port you set (use a free one if 8080 is taken)
   }
   ```

4. **Open the media port.** WHIP media is a single UDP port (`rtc_config.udp_port`, `7885` in the template); open it **directly** on the host firewall (UDP cannot go through the HTTP proxy). Set `rtc_config.use_external_ip: true` so the ingress advertises the host's public IP. The WHIP signalling port itself only needs to be reachable by your reverse proxy, not the public internet.

5. **Enable it in the app.** **Admin → Settings → Watch Parties, Voice & Streaming** → toggle **Enable Live Streaming**, and optionally set **Max Concurrent Streams** (default `3`). It reuses the same LiveKit URL + key/secret as voice, and does not need voice chat itself to be switched on.

Broadcasters then hit **Go live** on `/live`, paste the returned WHIP URL + stream key into OBS / Streamlabs (Service: **WHIP**), and appear on the public directory.

> WHIP media (WebRTC over UDP) requires the ingress on **host networking**. That works on a Linux host but not reliably on Docker Desktop (Windows/Mac), where the broadcaster app lives outside the container's network namespace, test broadcasting on the Linux host.

## Smooth Playback (LiveKit Egress / HLS)

By default a viewer watches a stream over **WebRTC**: sub-second latency, but it can freeze on a shaky connection, which is the nature of real-time UDP. Enabling the bundled `livekit-egress` service adds a second, **Smooth** option per stream, a buffered **HLS** rendition that plays a few seconds behind live but rides out network hiccups. Each viewer flips between **Low latency** (WebRTC) and **Smooth** (HLS) on the player, and the streamer picks which one viewers start on in the Go Live panel. It builds on the Ingress setup above. Disabled by default.

1. **Create the egress config.** The compose bind-mounts `./egress.yaml`, which is gitignored. Copy the template (otherwise Docker creates an empty directory in its place):

   ```bash
   cp egress.yaml.example egress.yaml
   ```

   Set `api_key` / `api_secret` / `ws_url` / `redis.address` to the **same** values as `ingress.yaml`. The egress must share the identical coordination plane or it never finds the room to record. In production (host networking) that is `ws://127.0.0.1:7880` and `127.0.0.1:<valkey-port>`.

2. **Where the files go.** The egress writes HLS segments to `stream_hls_output_dir` (default `/app/data/hls`), which the compose bind-mounts into the egress as `./data/hls` and into the app as part of `./data`. The app serves them at `/hls/*` on the **existing** site domain, exactly like `/uploads`, so there is **no new A record and no new reverse-proxy block**: your current Caddy/Nginx config already covers it. The app also owns the cleanup, removing each per-stream directory once the broadcast ends, so it needs write access to that path and not only read. The egress is outbound-only and receives no inbound media, so it opens **no new firewall ports** (unlike the ingress).

3. **Enable it in the app.** **Admin → Settings → Watch Parties, Voice & Streaming** → toggle **Enable Smooth (HLS) playback** (revealed when streaming is on). Leave **HLS Output Directory** at `/app/data/hls` unless you remap the mount.

The egress runs **participant egress** (no headless Chrome compositor), so it is light, but the container still needs `--cap-add=SYS_ADMIN` (already in the compose) because the egress image enables Chrome sandboxing regardless. When the broadcaster's video track appears, the app polls the ingress (up to six times, two seconds apart) for the **actual width / height / framerate** it is measuring from OBS and starts the egress at those, falling back to 60 fps if nothing has been reported yet, so the Smooth rendition mirrors whatever resolution the streamer set rather than a fixed one. The bitrate is not measured: the streamer types it into **Stream bitrate (Kbps)** in the Go Live panel, required whenever Smooth is available and accepted between 500 and 50000, and the egress encodes at that. Output is **H.264 Baseline** video with **320 kbps AAC** audio at 48 kHz, in two-second segments with a matching keyframe interval, for Safari/iOS compatibility. It is a **single rendition** (LiveKit egress encodes once per job, there is no adaptive-bitrate ladder), and the per-stream segment directory is deleted when the stream ends, with the reconcile pass sweeping up anything a crash orphaned, so nothing accumulates on disk.

> Smooth playback transcodes (HLS over `.ts` cannot carry the source codecs untouched), so it is a near-transparent re-encode at the bitrate the streamer entered, not bit-identical; the Low-latency WebRTC path stays the zero-loss option.

## Upgrading Postgres (18 to 19)

A new Postgres major cannot open the previous major's data. The image keeps each major in its own folder on the `/var/lib/postgresql` volume (`18/docker`, `19/docker`), and the 19 image refuses to start while it finds 18 data there and nothing in its own folder. Nothing is lost when that happens, but the database stays down until the data is moved across.

Move it with a dump and restore rather than `pg_upgrade`. A restore rebuilds every table and index on 19, so the stored `search_vector` columns are recomputed with 19's English stemmer (19 changed stems such as "pasted", "internal" and "interval"; `pg_upgrade` keeps the old ones, so searches for those words would miss older posts), long text is recompressed with 19's default `lz4` instead of `pglz`, and tables and indexes start with no bloat. The site is down for the dump plus the restore, so run the same steps against the dev compose first to see how long that takes.

Every merge to main deploys (`deploy.yml` runs `git pull`, `compose pull` and `up -d`). Merge nothing while this runs, and merge the `postgres:19` image line only after step 4: a deploy part-way through starts whichever image the compose file names against a half-moved volume.

Run everything in one shell from the deploy directory (on dev, set `C="docker compose"`). First check the database size and free disk: the volume needs room for a second copy of the database plus the dump file.

```bash
C="docker compose -f docker-compose.prod.yml"
PG=$($C ps -q postgres)
VOL=$(docker inspect "$PG" --format '{{range .Mounts}}{{if eq .Destination "/var/lib/postgresql"}}{{.Name}}{{end}}{{end}}')
docker exec "$PG" sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "SELECT pg_size_pretty(pg_database_size(current_database()))"'
df -h /var/lib/docker
```

1. **Stop the app** so nothing is written after the dump:

   ```bash
   $C stop umineko-city-of-books
   ```

2. **Dump with 19's `pg_dump`**, which the 19 release notes ask for, from the still-running 18 server:

   ```bash
   mkdir -p pgdump
   docker run --rm --network "container:$PG" --env-file postgres.env -v "$PWD/pgdump:/dump" postgres:19 \
     sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc -f /dump/pre-19.dump'
   ```

3. **Stop 18 and park its data** under a name the 19 image does not look for:

   ```bash
   $C stop postgres
   docker run --rm -v "$VOL:/var/lib/postgresql" postgres:19 mv /var/lib/postgresql/18/docker /var/lib/postgresql/18/pre-19
   ```

4. **Start 19 on the same volume and restore.** The first boot creates the user and database from `postgres.env`. `pg_isready` checks over TCP because first-boot setup runs a temporary socket-only server, and the TCP port only opens once setup is finished. `-j "$(nproc)"` follows the `pg_restore` docs, which suggest the number of CPU cores as the starting point.

   ```bash
   docker run -d --name pg19-restore --env-file postgres.env -v "$VOL:/var/lib/postgresql" -v "$PWD/pgdump:/dump:ro" postgres:19
   until docker exec pg19-restore pg_isready -h 127.0.0.1 -q; do sleep 1; done
   docker exec pg19-restore sh -c 'pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --exit-on-error -j "$(nproc)" /dump/pre-19.dump'
   docker exec pg19-restore sh -c 'vacuumdb -U "$POSTGRES_USER" -d "$POSTGRES_DB" --analyze-only'
   docker rm -f pg19-restore
   ```

5. **Merge the `postgres:19` image line.** The deploy pulls 19, starts it on the restored data and brings the app back. On dev, run `$C up -d`.

If a step fails before step 5, put 18 back. The box's compose file still names `postgres:18`, so `up -d` brings the site back as it was (on dev, set the image line back to 18 first):

```bash
docker rm -f pg19-restore
docker run --rm -v "$VOL:/var/lib/postgresql" postgres:19 sh -c 'rm -rf /var/lib/postgresql/19 && mv /var/lib/postgresql/18/pre-19 /var/lib/postgresql/18/docker'
$C up -d
```

Once the site is working on 19, delete the parked 18 data and the dump:

```bash
docker run --rm -v "$VOL:/var/lib/postgresql" postgres:19 rm -rf /var/lib/postgresql/18
rm -rf pgdump
```
