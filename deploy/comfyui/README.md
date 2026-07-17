# ComfyUI GPU Docker Deployment

This image is used by `deploy/docker-compose.single.yml` and is tuned for NVIDIA GPU acceleration.

## Start

From the repository root:

```bash
docker compose -f deploy/docker-compose.single.yml up --build -d comfyui
```

Open:

```text
http://localhost:8188
```

## GPU

The compose service uses `gpus: all` and the NVIDIA container runtime. The image installs CUDA 12.8 PyTorch wheels by default, suitable for newer NVIDIA cards such as RTX 50 series.

Optional `.env` values:

```env
COMFYUI_PORT=8188
COMFYUI_REF=master
PYTORCH_CUDA_INDEX=https://download.pytorch.org/whl/cu128
```

## Persistent Data

The compose service keeps these Docker volumes:

- `comfyui_models` -> `/app/ComfyUI/models`
- `comfyui_custom_nodes` -> `/app/ComfyUI/custom_nodes`
- `comfyui_input` -> `/app/ComfyUI/input`
- `comfyui_output` -> `/app/ComfyUI/output`
- `comfyui_user` -> `/app/ComfyUI/user`

Put checkpoints under `models/checkpoints`. IP-Adapter, ControlNet, LoRA and VAE files should go into their matching ComfyUI model folders.
