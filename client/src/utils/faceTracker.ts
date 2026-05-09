import type { FaceBox } from '../components/FaceOverlay'

export interface Track {
  id: number
  box: FaceBox
  name?: string
  lastSeen: number
  age: number
  unmatchedFrames: number
}

export class FaceTracker {
  private tracks: Track[] = []
  private nextId = 1

  update(detections: FaceBox[], now: number): { tracks: Track[]; newTrackIds: number[]; newTrackDetIdx: Map<number, number> } {
    const n = this.tracks.length
    const m = detections.length
    const iouMatrix: number[][] = Array.from({ length: n }, () => Array(m).fill(0))

    for (let i = 0; i < n; i++) {
      for (let j = 0; j < m; j++) {
        iouMatrix[i][j] = this.iou(this.tracks[i].box, detections[j])
      }
    }

    const allPairs: [number, number, number][] = []
    for (let i = 0; i < n; i++) {
      for (let j = 0; j < m; j++) {
        if (iouMatrix[i][j] > 0.3) allPairs.push([i, j, iouMatrix[i][j]])
      }
    }
    allPairs.sort((a, b) => b[2] - a[2])

    const trackMatched = new Set<number>()
    const detMatched = new Set<number>()

    for (const [ti, di] of allPairs) {
      if (!trackMatched.has(ti) && !detMatched.has(di)) {
        trackMatched.add(ti)
        detMatched.add(di)
        this.tracks[ti].box = detections[di]
        this.tracks[ti].lastSeen = now
        this.tracks[ti].age++
        this.tracks[ti].unmatchedFrames = 0
      }
    }

    const newTrackIds: number[] = []
    const newTrackDetIdx = new Map<number, number>()
    for (let j = 0; j < m; j++) {
      if (!detMatched.has(j)) {
        const id = this.nextId++
        this.tracks.push({
          id,
          box: detections[j],
          lastSeen: now,
          age: 1,
          unmatchedFrames: 0,
        })
        newTrackIds.push(id)
        newTrackDetIdx.set(id, j)
      }
    }

    for (let i = 0; i < this.tracks.length; i++) {
      if (!trackMatched.has(i)) this.tracks[i].unmatchedFrames++
    }

    this.tracks = this.tracks.filter((t) => t.unmatchedFrames < 2)

    return {
      tracks: this.tracks.filter((t) => t.age >= 2),
      newTrackIds,
      newTrackDetIdx,
    }
  }

  setName(trackId: number, name: string | undefined) {
    const t = this.tracks.find((t) => t.id === trackId)
    if (t) t.name = name
  }

  reset() {
    this.tracks = []
  }

  private iou(a: FaceBox, b: FaceBox): number {
    const ax1 = a.x, ay1 = a.y, ax2 = a.x + a.width, ay2 = a.y + a.height
    const bx1 = b.x, by1 = b.y, bx2 = b.x + b.width, by2 = b.y + b.height
    const ix1 = Math.max(ax1, bx1)
    const iy1 = Math.max(ay1, by1)
    const ix2 = Math.min(ax2, bx2)
    const iy2 = Math.min(ay2, by2)
    if (ix2 <= ix1 || iy2 <= iy1) return 0
    const inter = (ix2 - ix1) * (iy2 - iy1)
    const areaA = (ax2 - ax1) * (ay2 - ay1)
    const areaB = (bx2 - bx1) * (by2 - by1)
    return inter / (areaA + areaB - inter)
  }
}
