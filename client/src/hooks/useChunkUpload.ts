import { useState, useRef, useCallback } from 'react'
import * as fileApi from '../services/file'
import * as adminApi from '../services/admin'

const CHUNK_SIZE = 10 * 1024 * 1024 // 10MB

export type UploadMode = 'sequential' | 'concurrent'

export interface ChunkProgress {
  completed: number
  total: number
  percent: number
  currentChunk: number
}

interface ActiveUpload {
  uploadId: string
  fileName: string
  fileSize: number
  totalChunks: number
  abort: AbortController
}

export default function useChunkUpload() {
  const [active, setActive] = useState<ActiveUpload | null>(null)
  const [progress, setProgress] = useState<ChunkProgress>({ completed: 0, total: 0, percent: 0, currentChunk: -1 })
  const [error, setError] = useState<string | null>(null)
  const uploadingRef = useRef(false)

  const getUploadConfig = useCallback(async () => {
    try {
      const configs = await adminApi.getSystemConfig()
      const seq = configs.find((c) => c.key === 'upload.sequential_mode')
      const max = configs.find((c) => c.key === 'upload.max_concurrent_chunks')
      return {
        sequential: seq?.value === 'true',
        maxConcurrent: parseInt(max?.value || '3', 10),
      }
    } catch {
      return { sequential: false, maxConcurrent: 3 }
    }
  }, [])

  const startUpload = useCallback(async (
    file: File,
    parentId: string,
    onComplete?: (item: fileApi.FileItem) => void,
  ) => {
    if (uploadingRef.current) return
    uploadingRef.current = true
    setError(null)

    try {
      const totalChunks = Math.ceil(file.size / CHUNK_SIZE)
      // Init
      const init = await fileApi.initChunkUpload({
        file_name: file.name,
        file_size: file.size,
        parent_id: parentId,
        mime_type: file.type || undefined,
      })

      const uploadId = init.upload_id
      const abort = new AbortController()
      setActive({ uploadId, fileName: file.name, fileSize: file.size, totalChunks, abort })
      setProgress({ completed: 0, total: totalChunks, percent: 0, currentChunk: -1 })

      const config = await getUploadConfig()

      const completedSet = new Set<number>()
      const updateProgress = (chunkIdx: number) => {
        completedSet.add(chunkIdx)
        const done = completedSet.size
        setProgress({ completed: done, total: totalChunks, percent: Math.round((done / totalChunks) * 100), currentChunk: chunkIdx })
      }

      if (config.sequential) {
        for (let i = 0; i < totalChunks; i++) {
          const start = i * CHUNK_SIZE
          const end = Math.min(start + CHUNK_SIZE, file.size)
          const blob = file.slice(start, end)
          await fileApi.uploadChunk(uploadId, i, blob)
          updateProgress(i)
        }
      } else {
        // Concurrent with limit
        const maxConcurrent = config.maxConcurrent
        const pending: number[] = []
        for (let i = 0; i < totalChunks; i++) pending.push(i)

        let running = 0
        const runChunk = async (chunkIndex: number): Promise<void> => {
          const start = chunkIndex * CHUNK_SIZE
          const end = Math.min(start + CHUNK_SIZE, file.size)
          const blob = file.slice(start, end)
          await fileApi.uploadChunk(uploadId, chunkIndex, blob)
          updateProgress(chunkIndex)
        }

        const workers: Promise<void>[] = []
        const next = async () => {
          while (pending.length > 0) {
            const idx = pending.shift()!
            running++
            await runChunk(idx)
            running--
          }
        }

        for (let w = 0; w < maxConcurrent; w++) {
          workers.push(next())
        }
        await Promise.all(workers)
      }

      const result = await fileApi.completeChunkUpload(uploadId)
      setActive(null)
      setProgress({ completed: totalChunks, total: totalChunks, percent: 100, currentChunk: -1 })
      onComplete?.(result)
      return result
    } catch (e: any) {
      setError(e.response?.data?.message || e.message || '上传失败')
      throw e
    } finally {
      uploadingRef.current = false
    }
  }, [getUploadConfig])

  const resumeUpload = useCallback(async (
    file: File,
    _parentId: string,
    uploadId: string,
    totalChunks: number,
    completedChunks: number[],
    onComplete?: (item: fileApi.FileItem) => void,
  ) => {
    if (uploadingRef.current) return
    uploadingRef.current = true
    setError(null)

    try {
      const abort = new AbortController()
      setActive({ uploadId, fileName: file.name, fileSize: file.size, totalChunks, abort })

      const completedSet = new Set<number>(completedChunks)
      setProgress({ completed: completedSet.size, total: totalChunks, percent: Math.round((completedSet.size / totalChunks) * 100), currentChunk: -1 })

      const config = await getUploadConfig()

      const updateProgress = (chunkIdx: number) => {
        completedSet.add(chunkIdx)
        const done = completedSet.size
        setProgress({ completed: done, total: totalChunks, percent: Math.round((done / totalChunks) * 100), currentChunk: chunkIdx })
      }

      // Only upload missing chunks
      const missing: number[] = []
      for (let i = 0; i < totalChunks; i++) {
        if (!completedSet.has(i)) missing.push(i)
      }

      if (config.sequential) {
        for (const i of missing) {
          const start = i * CHUNK_SIZE
          const end = Math.min(start + CHUNK_SIZE, file.size)
          const blob = file.slice(start, end)
          await fileApi.uploadChunk(uploadId, i, blob)
          updateProgress(i)
        }
      } else {
        const maxConcurrent = config.maxConcurrent
        const pending = [...missing]
        let running = 0

        const runChunk = async (chunkIndex: number): Promise<void> => {
          const start = chunkIndex * CHUNK_SIZE
          const end = Math.min(start + CHUNK_SIZE, file.size)
          const blob = file.slice(start, end)
          await fileApi.uploadChunk(uploadId, chunkIndex, blob)
          updateProgress(chunkIndex)
        }

        const workers: Promise<void>[] = []
        const next = async () => {
          while (pending.length > 0) {
            const idx = pending.shift()!
            running++
            await runChunk(idx)
            running--
          }
        }

        for (let w = 0; w < Math.min(maxConcurrent, missing.length); w++) {
          workers.push(next())
        }
        await Promise.all(workers)
      }

      const result = await fileApi.completeChunkUpload(uploadId)
      setActive(null)
      setProgress({ completed: totalChunks, total: totalChunks, percent: 100, currentChunk: -1 })
      onComplete?.(result)
      return result
    } catch (e: any) {
      setError(e.response?.data?.message || e.message || '上传失败')
      throw e
    } finally {
      uploadingRef.current = false
    }
  }, [getUploadConfig])

  const cancel = useCallback(async () => {
    if (active) {
      try {
        await fileApi.cancelChunkUpload(active.uploadId)
      } catch { /* ignore */ }
      setActive(null)
      uploadingRef.current = false
    }
  }, [active])

  const reset = useCallback(() => {
    setActive(null)
    setProgress({ completed: 0, total: 0, percent: 0, currentChunk: -1 })
    setError(null)
    uploadingRef.current = false
  }, [])

  return {
    active,
    progress,
    error,
    uploading: uploadingRef.current,
    startUpload,
    resumeUpload,
    cancel,
    reset,
  }
}
