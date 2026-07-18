FROM python:3.12-slim

ARG COMFYUI_REF=master
ARG PYTORCH_CUDA_INDEX=https://download.pytorch.org/whl/cu128

ENV DEBIAN_FRONTEND=noninteractive \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PATH=/opt/venv/bin:$PATH

WORKDIR /app

RUN sed -i 's/ trixie-updates//g' /etc/apt/sources.list.d/debian.sources \
    && apt-get -o Acquire::Retries=5 update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        ffmpeg \
        git \
        libgl1 \
        libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

RUN python -m venv /opt/venv \
    && pip install --upgrade pip setuptools wheel

RUN git clone --depth 1 --branch "${COMFYUI_REF}" https://github.com/comfyanonymous/ComfyUI.git

WORKDIR /app/ComfyUI

RUN pip install --index-url "${PYTORCH_CUDA_INDEX}" torch torchvision torchaudio \
    && grep -Ev '^(comfyui-workflow-templates|comfyui-embedded-docs)' requirements.txt > /tmp/requirements-runtime.txt \
    && pip install -r /tmp/requirements-runtime.txt

RUN pip install comfyui-workflow-templates==0.11.9 comfyui-embedded-docs==0.5.8

EXPOSE 8188

ENTRYPOINT ["python", "main.py"]
