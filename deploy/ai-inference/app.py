import os
import torch
from ultralytics import YOLO
from fastapi import FastAPI, UploadFile, File
import uvicorn

app = FastAPI(title="CloudNexus AI Inference")

# Auto-detect device
DEVICE = "cuda" if torch.cuda.is_available() else "cpu"
MODEL_PATH = "/app/yolov8n.pt"

print(f"[inference] device={DEVICE}, model={MODEL_PATH}")

model = YOLO(MODEL_PATH)
model.to(DEVICE)


@app.get("/health")
def health():
    return {
        "status": "ok",
        "device": DEVICE,
        "model": MODEL_PATH,
        "cuda_available": torch.cuda.is_available(),
    }


@app.post("/detect")
async def detect(image: UploadFile = File(...)):
    contents = await image.read()
    # Save to temp file for YOLO (it accepts file paths)
    tmp_path = f"/tmp/{image.filename or 'frame.jpg'}"
    with open(tmp_path, "wb") as f:
        f.write(contents)

    results = model(tmp_path, device=DEVICE, verbose=False)
    os.remove(tmp_path)

    objects = []
    for r in results:
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

    return {"objects": objects, "error": ""}


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
