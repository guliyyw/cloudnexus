FROM python:3.11-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libxcb1 libx11-6 libgl1 libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir \
    torch torchvision \
    --index-url https://download.pytorch.org/whl/cpu

RUN pip install --no-cache-dir ultralytics fastapi uvicorn python-multipart

COPY ai-inference/app.py /app/app.py
COPY ai-inference/yolov8n.pt /app/yolov8n.pt
WORKDIR /app

CMD ["python", "/app/app.py"]
