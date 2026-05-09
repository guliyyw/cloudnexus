import * as faceapi from 'face-api.js'

const MODEL_URL = '/models'

let loaded = false

export async function loadModels(): Promise<void> {
  if (loaded) return
  await Promise.all([
    faceapi.nets.tinyFaceDetector.loadFromUri(MODEL_URL),
    faceapi.nets.faceLandmark68Net.loadFromUri(MODEL_URL),
    faceapi.nets.faceRecognitionNet.loadFromUri(MODEL_URL),
  ])
  loaded = true
}

export interface DetectedFace {
  descriptor: Float32Array
  box: { x: number; y: number; width: number; height: number }
}

export async function detectFaces(
  input: HTMLVideoElement | HTMLCanvasElement | HTMLImageElement,
): Promise<DetectedFace[]> {
  if (!loaded) await loadModels()

  const result = await faceapi
    .detectAllFaces(input, new faceapi.TinyFaceDetectorOptions({ inputSize: 320, scoreThreshold: 0.5 }))
    .withFaceLandmarks()
    .withFaceDescriptors()

  return result.map((r) => ({
    descriptor: r.descriptor,
    box: {
      x: r.detection.box.x,
      y: r.detection.box.y,
      width: r.detection.box.width,
      height: r.detection.box.height,
    },
  }))
}

export function embeddingToArray(descriptor: Float32Array): number[] {
  return Array.from(descriptor)
}
