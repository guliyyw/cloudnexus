FROM python:3.12-slim

ARG COMFYUI_REF=master
ARG PYTORCH_CUDA_INDEX=https://mirrors.aliyun.com/pytorch-wheels/cu130
ARG PYTORCH_VERSION=2.11.0
ARG TORCHVISION_VERSION=0.26.0
ARG TORCHAUDIO_VERSION=2.11.0

ENV DEBIAN_FRONTEND=noninteractive \
    PYTHONUNBUFFERED=1 \
    PIP_INDEX_URL=https://mirrors.aliyun.com/pypi/simple \
    PATH=/opt/venv/bin:$PATH

WORKDIR /app

RUN sed -i \
    -e 's#http://deb.debian.org/debian-security#https://mirrors.aliyun.com/debian-security#g' \
    -e 's#http://deb.debian.org/debian#https://mirrors.aliyun.com/debian#g' \
    -e 's/ trixie-updates//g' /etc/apt/sources.list.d/debian.sources \
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

RUN --mount=type=cache,target=/root/.cache/pip pip install --retries 10 --timeout 120 \
        "${PYTORCH_CUDA_INDEX}/torch-${PYTORCH_VERSION}%2Bcu130-cp312-cp312-manylinux_2_28_x86_64.whl" \
        "${PYTORCH_CUDA_INDEX}/torchvision-${TORCHVISION_VERSION}%2Bcu130-cp312-cp312-manylinux_2_28_x86_64.whl" \
        "${PYTORCH_CUDA_INDEX}/torchaudio-${TORCHAUDIO_VERSION}%2Bcu130-cp312-cp312-manylinux_2_28_x86_64.whl" \
    && pip install --retries 10 --timeout 120 -r requirements.txt

RUN --mount=type=cache,target=/root/.cache/pip apt-get -o Acquire::Retries=5 update \
    && apt-get install -y --no-install-recommends build-essential \
    && pip install --retries 10 --timeout 120 insightface==0.7.3 onnxruntime \
    && apt-get purge -y --auto-remove build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /usr/local/bin/comfyui-entrypoint
RUN sed -i 's/\r$//' /usr/local/bin/comfyui-entrypoint \
    && chmod +x /usr/local/bin/comfyui-entrypoint

EXPOSE 8188

ENTRYPOINT ["/usr/local/bin/comfyui-entrypoint"]
