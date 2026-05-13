import os
import time
import uuid
import torch
import cv2
from ultralytics import YOLO
from fastapi import FastAPI, UploadFile, File, Query, HTTPException, Header
import uvicorn

app = FastAPI(title="CloudNexus AI Inference")

# Shared secret for internal service-to-service auth
AUTH_TOKEN = os.getenv("AI_INFERENCE_TOKEN", "")

# Auto-detect device
DEVICE = "cuda" if torch.cuda.is_available() else "cpu"
MODEL_PATH = "/app/yolov8n.pt"

print(f"[inference] device={DEVICE}, model={MODEL_PATH}")

model = YOLO(MODEL_PATH)
model.to(DEVICE)


def extract_objects(r) -> list[dict]:
    """Extract filtered detections from a YOLO result."""
    objects = []
    for box in r.boxes:
        x1, y1, x2, y2 = box.xyxy[0].tolist()
        conf = float(box.conf[0])
        cls_id = int(box.cls[0])
        cls_name = model.names.get(cls_id, str(cls_id))
        if conf >= 0.5:
            objects.append({
                "class": cls_name,
                "confidence": round(conf, 4),
                "x1": round(x1, 1),
                "y1": round(y1, 1),
                "x2": round(x2, 1),
                "y2": round(y2, 1),
            })
    return objects


@app.get("/health")
def health():
    return {
        "status": "ok",
        "device": DEVICE,
        "model": MODEL_PATH,
        "cuda_available": torch.cuda.is_available(),
    }


@app.post("/detect")
async def detect(image: UploadFile = File(...), authorization: str = Header(default="", alias="Authorization")):
    if AUTH_TOKEN and authorization != f"Bearer {AUTH_TOKEN}":
        raise HTTPException(status_code=401, detail="Unauthorized")
    contents = await image.read()
    tmp_path = f"/tmp/{image.filename or 'frame.jpg'}"
    with open(tmp_path, "wb") as f:
        f.write(contents)

    results = model(tmp_path, device=DEVICE, verbose=False)
    os.remove(tmp_path)

    objects = []
    for r in results:
        objects = extract_objects(r)

    return {"objects": objects, "error": ""}


@app.post("/detect-video")
async def detect_video(
    video: UploadFile = File(...),
    interval: float = Query(2.0, ge=0.5, le=60.0, description="Seconds between sampled frames"),
    authorization: str = Header(default="", alias="Authorization"),
):
    if AUTH_TOKEN and authorization != f"Bearer {AUTH_TOKEN}":
        raise HTTPException(status_code=401, detail="Unauthorized")

    # Save uploaded video to temp file
    ext = os.path.splitext(video.filename or "video.mp4")[1] or ".mp4"
    tmp_path = f"/tmp/{uuid.uuid4().hex}{ext}"
    with open(tmp_path, "wb") as f:
        f.write(await video.read())

    cap = cv2.VideoCapture(tmp_path)
    if not cap.isOpened():
        os.remove(tmp_path)
        return {"error": "无法打开视频文件", "detections": []}

    fps = cap.get(cv2.CAP_PROP_FPS)
    total_frames = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
    duration = total_frames / fps if fps > 0 else 0
    frame_interval = max(1, int(fps * interval))

    detections = []
    frames_analyzed = 0
    frame_idx = 0
    t0 = time.time()

    while True:
        ret, frame = cap.read()
        if not ret:
            break

        if frame_idx % frame_interval == 0:
            # Save frame to temp image for YOLO
            frame_path = f"/tmp/frame_{frame_idx}.jpg"
            cv2.imwrite(frame_path, frame)

            results = model(frame_path, device=DEVICE, verbose=False)
            objects = extract_objects(results[0]) if results else []
            os.remove(frame_path)

            frames_analyzed += 1
            if objects:
                detections.append({
                    "time": round(frame_idx / fps, 1),
                    "objects": objects,
                })

        frame_idx += 1

    cap.release()
    os.remove(tmp_path)

    elapsed = round(time.time() - t0, 1)
    print(f"[detect-video] duration={duration:.1f}s, fps={fps:.1f}, "
          f"frames_analyzed={frames_analyzed}, detections={len(detections)}, elapsed={elapsed}s")

    return {
        "error": "",
        "video_duration": round(duration, 1),
        "fps": round(fps, 1),
        "frames_analyzed": frames_analyzed,
        "detections": detections,
    }


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
