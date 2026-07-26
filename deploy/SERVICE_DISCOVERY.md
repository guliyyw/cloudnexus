# Service discovery and selective startup

## Default startup

`docker-compose.single.yml` now starts a minimal stack by default:

- `postgres`
- `redis`
- `minio`
- `user-file-svc`
- `nginx`

Start it with:

```bash
docker compose -f docker-compose.single.yml up -d
```

Nginx starts independently from optional services. If a route points to a service
that is not running yet, nginx returns `503` instead of failing to start.

## Optional modules

Start modules by profile:

```bash
docker compose -f docker-compose.single.yml --profile im up -d
docker compose -f docker-compose.single.yml --profile docker up -d
docker compose -f docker-compose.single.yml --profile camera up -d
docker compose -f docker-compose.single.yml --profile collab up -d
docker compose -f docker-compose.single.yml --profile drama up -d
docker compose -f docker-compose.single.yml --profile comfyui up -d
```

Start a single service directly:

```bash
docker compose -f docker-compose.single.yml up -d drama-svc
docker compose -f docker-compose.single.yml up -d comfyui
```

Stop an optional module:

```bash
docker compose -f docker-compose.single.yml stop drama-svc comfyui
```

## Runtime discovery

Nginx uses Docker DNS (`127.0.0.11`) and resolves backend service names at
request time. This allows:

- nginx to start before optional services
- optional services to be started later
- service containers to be recreated without restarting nginx

## Cross-server preparation

Runtime backend targets are variables in `nginx.conf`. Files ending with `.conf`
under `deploy/nginx/discovery/` are included after the defaults.

A future discovery controller can generate a file like:

```nginx
set $user_file_backend "http://10.0.0.11:8081";
set $im_backend "http://10.0.0.12:8082";
set $drama_backend "http://10.0.0.13:8087";
```

Then reload nginx after the file changes:

```bash
docker compose -f docker-compose.single.yml exec nginx nginx -s reload
```

For full no-reload cross-server routing, the next step should be a dynamic
gateway such as Traefik with Docker/Consul providers, or Nginx backed by a local
DNS service that resolves stable backend names.
