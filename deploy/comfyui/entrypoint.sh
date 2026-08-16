#!/bin/sh
set -eu

download_file() {
  url="$1"
  target="$2"
  if [ -s "$target" ]; then
    return
  fi
  mkdir -p "$(dirname "$target")"
  temp="${target}.part"
  echo "Downloading $(basename "$target")..."
  curl --fail --location --retry 5 --retry-delay 3 --continue-at - --output "$temp" "$url"
  mv "$temp" "$target"
}

if [ "${COMFYUI_DOWNLOAD_FACEID_MODELS:-true}" = "true" ]; then
  download_file \
    "https://huggingface.co/h94/IP-Adapter-FaceID/resolve/main/ip-adapter-faceid-plusv2_sdxl.bin" \
    "/app/ComfyUI/models/ipadapter/ip-adapter-faceid-plusv2_sdxl.bin"
  download_file \
    "https://huggingface.co/h94/IP-Adapter-FaceID/resolve/main/ip-adapter-faceid-plusv2_sdxl_lora.safetensors" \
    "/app/ComfyUI/models/loras/ip-adapter-faceid-plusv2_sdxl_lora.safetensors"

  insightface_dir="/root/.insightface/models/buffalo_l"
  if [ ! -s "${insightface_dir}/w600k_r50.onnx" ]; then
    archive="/tmp/buffalo_l.zip"
    download_file \
      "https://github.com/deepinsight/insightface/releases/download/v0.7/buffalo_l.zip" \
      "$archive"
    mkdir -p "$insightface_dir"
    python -c 'import sys,zipfile; zipfile.ZipFile(sys.argv[1]).extractall(sys.argv[2])' "$archive" "$insightface_dir"
    rm -f "$archive"
  fi
fi

if [ "${COMFYUI_DOWNLOAD_REALVISXL:-true}" = "true" ]; then
  download_file \
    "https://huggingface.co/SG161222/RealVisXL_V5.0/resolve/main/RealVisXL_V5.0_fp16.safetensors" \
    "/app/ComfyUI/models/checkpoints/RealVisXL_V5.0_fp16.safetensors"
fi

# MiniMax H3 is opt-in because the four open-weight files are large.  The
# model directory is persistent, so enabling this once is enough.
if [ "${COMFYUI_DOWNLOAD_H3_MODELS:-false}" = "true" ]; then
  h3_repo="https://huggingface.co/Comfy-Org/MiniMax-H3/resolve/main"
  download_file "${h3_repo}/minimax_h3_fl2va_pruned_int8_convrot.safetensors" "/app/ComfyUI/models/diffusion_models/minimax_h3_fl2va_pruned_int8_convrot.safetensors"
  download_file "${h3_repo}/minimax_h3_ref2va_pruned_int8_convrot.safetensors" "/app/ComfyUI/models/diffusion_models/minimax_h3_ref2va_pruned_int8_convrot.safetensors"
  download_file "${h3_repo}/qwen3vl_32b_minimax_h3_nvfp4_awq.safetensors" "/app/ComfyUI/models/text_encoders/qwen3vl_32b_minimax_h3_nvfp4_awq.safetensors"
  download_file "${h3_repo}/minimax_h3_video_vae_fp16.safetensors" "/app/ComfyUI/models/vae/minimax_h3_video_vae_fp16.safetensors"
  download_file "${h3_repo}/minimax_h3_audio_vae_fp32.safetensors" "/app/ComfyUI/models/vae/minimax_h3_audio_vae_fp32.safetensors"
fi

exec python main.py "$@"
