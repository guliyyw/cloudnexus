export class VideoRecorder {
  private canvas: HTMLCanvasElement | null = null
  private ctx: CanvasRenderingContext2D | null = null
  private mediaRecorder: MediaRecorder | null = null
  private chunks: Blob[] = []
  private animId = 0
  private _recording = false
  private maxChunks = 60 // 5 minutes at 5s intervals

  get recording() { return this._recording }

  start(video: HTMLVideoElement): void {
    if (this._recording) return
    this._setup(video.videoWidth || 640, video.videoHeight || 360)
    const draw = () => {
      if (!this._recording) return
      if (video.readyState >= 2) this.ctx!.drawImage(video, 0, 0)
      this.animId = requestAnimationFrame(draw)
    }
    this._recording = true
    this.animId = requestAnimationFrame(draw)
  }

  startFromCanvas(source: HTMLCanvasElement): void {
    if (this._recording) return
    this._setup(source.width || 640, source.height || 360)
    const draw = () => {
      if (!this._recording) return
      if (source.width > 0) this.ctx!.drawImage(source, 0, 0)
      this.animId = requestAnimationFrame(draw)
    }
    this._recording = true
    this.animId = requestAnimationFrame(draw)
  }

  private _setup(w: number, h: number): void {
    this.canvas = document.createElement('canvas')
    this.canvas.width = w
    this.canvas.height = h
    this.ctx = this.canvas.getContext('2d')!
    this.chunks = []
    const stream = this.canvas.captureStream(30)
    this.mediaRecorder = new MediaRecorder(stream, {
      mimeType: MediaRecorder.isTypeSupported('video/webm;codecs=vp9')
        ? 'video/webm;codecs=vp9'
        : 'video/webm',
    })
    this.mediaRecorder.ondataavailable = (e) => {
      if (e.data.size > 0) {
        this.chunks.push(e.data)
        if (this.chunks.length > this.maxChunks) {
          URL.revokeObjectURL(URL.createObjectURL(this.chunks.shift()!))
        }
      }
    }
    this.mediaRecorder.start(5000)
  }

  stop(): Blob | null {
    this._recording = false
    cancelAnimationFrame(this.animId)

    if (this.mediaRecorder && this.mediaRecorder.state !== 'inactive') {
      this.mediaRecorder.stop()
    }

    const blob = this.getRecordedBlob()
    this.chunks = []
    this.canvas = null
    this.ctx = null
    this.mediaRecorder = null
    return blob
  }

  getRecordedBlob(): Blob {
    return new Blob(this.chunks, { type: 'video/webm' })
  }

  getBlobUrl(): string {
    return URL.createObjectURL(this.getRecordedBlob())
  }

  getDurationMs(): number {
    // rough estimate: each chunk is ~5s, so total duration ≈ chunks * 5000
    return this.chunks.length * 5000
  }

  destroy(): void {
    this.stop()
    this.chunks.forEach((c) => URL.revokeObjectURL(URL.createObjectURL(c)))
    this.chunks = []
  }
}
