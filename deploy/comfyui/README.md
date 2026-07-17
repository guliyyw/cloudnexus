# ComfyUI Docker Deployment

This directory contains the Docker build used by `deploy/docker-compose.single.yml`.

## Start

From `deploy/`:

```bash
docker compose -f docker-compose.single.yml up --build -d comfyui
```

Open:

```text
http://localhost:8188
```

## Configuration

Optional `.env` values:

```env
COMFYUI_PORT=8188
COMFYUI_REF=master
```

`COMFYUI_REF` can be changed to a ComfyUI branch or tag before rebuilding.

## Persistent Data

The compose service keeps these Docker volumes:

- `comfyui_models` -> `/app/ComfyUI/models`
- `comfyui_custom_nodes` -> `/app/ComfyUI/custom_nodes`
- `comfyui_input` -> `/app/ComfyUI/input`
- `comfyui_output` -> `/app/ComfyUI/output`
- `comfyui_user` -> `/app/ComfyUI/user`

To add checkpoints, LoRAs, VAEs, or custom nodes, copy them into the matching volume or mount a host directory instead.

## NVIDIA GPU

The default image installs CPU PyTorch so it can run everywhere. For NVIDIA GPU acceleration, switch the PyTorch install line in `Dockerfile` to a CUDA wheel index that matches your host driver, then run Docker with NVIDIA Container Toolkit enabled.
