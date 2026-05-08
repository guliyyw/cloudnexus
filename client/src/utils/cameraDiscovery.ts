// Browser-side LAN camera discovery.
// Probes the client's local subnet for HTTP/ONVIF cameras using fetch().
// Unlike server-side scanning, this works when the browser and cameras
// are on the same LAN, regardless of where the server is deployed.

export interface LocalDiscoveredCamera {
  ip: string
  port: number
  path: string
  source: string // "http" | "onvif"
}

const CAMERA_PATHS = [
  { port: 80, path: '/onvif/device_service' },
  { port: 80, path: '/' },
  { port: 80, path: '/snapshot.jpg' },
  { port: 8080, path: '/onvif/device_service' },
  { port: 8080, path: '/' },
  { port: 8080, path: '/snapshot.jpg' },
  { port: 8000, path: '/onvif/device_service' },
  { port: 8000, path: '/' },
]

const MAX_CONCURRENT = 50
const PROBE_TIMEOUT_MS = 2000

// detectLocalIP uses WebRTC to find the browser's local IP address.
export async function detectLocalIP(): Promise<string> {
  return new Promise((resolve) => {
    const pc = new RTCPeerConnection({
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
    })
    const timeout = setTimeout(() => {
      pc.close()
      resolve('')
    }, 3000)

    pc.createDataChannel('')
    pc.createOffer().then((offer) => pc.setLocalDescription(offer))

    pc.onicecandidate = (e) => {
      if (!e.candidate) return
      const addr = extractIP(e.candidate.candidate || '')
      if (addr) {
        clearTimeout(timeout)
        pc.close()
        resolve(addr)
      }
    }

    pc.onicegatheringstatechange = () => {
      if (pc.iceGatheringState === 'complete') {
        // If we haven't resolved yet, try from local description
        const sdp = pc.localDescription?.sdp || ''
        const ip = extractIPFromSDP(sdp)
        if (ip) {
          clearTimeout(timeout)
          pc.close()
          resolve(ip)
        }
      }
    }
  })
}

function extractIP(candidate: string): string {
  const m = candidate.match(/(\d+\.\d+\.\d+\.\d+)/)
  if (!m) return ''
  const ip = m[1]
  if (ip.startsWith('127.') || ip.startsWith('0.')) return ''
  return ip
}

function extractIPFromSDP(sdp: string): string {
  const m = sdp.match(/c=IN IP4 (\d+\.\d+\.\d+\.\d+)/)
  if (!m) return ''
  return m[1]
}

// generateSubnetIPs creates all IPs in a /24 subnet from the given IP.
export function generateSubnetIPs(localIP: string): string[] {
  const parts = localIP.split('.')
  if (parts.length !== 4) return []
  const base = `${parts[0]}.${parts[1]}.${parts[2]}`
  const ips: string[] = []
  for (let i = 1; i <= 254; i++) {
    ips.push(`${base}.${i}`)
  }
  return ips
}

async function probeHost(
  ip: string,
  port: number,
  path: string,
): Promise<LocalDiscoveredCamera | null> {
  const url = `http://${ip}:${port}${path}`
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), PROBE_TIMEOUT_MS)

  try {
    await fetch(url, {
      mode: 'no-cors',
      signal: controller.signal,
    })
    clearTimeout(timer)
    const source = path.includes('onvif') ? 'onvif' : 'http'
    return { ip, port, path, source }
  } catch {
    clearTimeout(timer)
    return null
  }
}

// discoverCamerasLocally scans the given subnet from the browser.
// Returns discovered cameras and calls onProgress with {scanned, total}.
export async function discoverCamerasLocally(
  subnetIPs: string[],
  onProgress?: (scanned: number, total: number) => void,
): Promise<LocalDiscoveredCamera[]> {
  const results: LocalDiscoveredCamera[] = []
  const seen = new Set<string>()
  let completed = 0
  const total = subnetIPs.length * CAMERA_PATHS.length

  // Build all probe tasks
  const tasks: Array<() => Promise<void>> = []
  for (const ip of subnetIPs) {
    for (const { port, path } of CAMERA_PATHS) {
      tasks.push(async () => {
        const result = await probeHost(ip, port, path)
        completed++
        if (result) {
          const key = `${result.ip}:${result.port}`
          if (!seen.has(key)) {
            seen.add(key)
            results.push(result)
          }
        }
        if (completed % 50 === 0 || completed === total) {
          onProgress?.(completed, total)
        }
      })
    }
  }

  // Run with concurrency limit
  let idx = 0
  async function worker() {
    while (idx < tasks.length) {
      const i = idx++
      await tasks[i]()
    }
  }

  const workers = Array.from(
    { length: Math.min(MAX_CONCURRENT, tasks.length) },
    () => worker(),
  )
  await Promise.all(workers)

  // Final progress
  onProgress?.(completed, total)
  return results
}

// subnetFromIP extracts /24 subnet CIDR from an IP.
export function subnetFromIP(ip: string): string {
  const parts = ip.split('.')
  if (parts.length !== 4) return '192.168.1.0/24'
  return `${parts[0]}.${parts[1]}.${parts[2]}.0/24`
}
