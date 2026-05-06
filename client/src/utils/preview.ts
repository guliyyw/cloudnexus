export function isPreviewable(mime: string): boolean {
  return mime?.startsWith('image/') || mime?.startsWith('video/') ||
         mime?.startsWith('audio/') || mime === 'application/pdf'
}
