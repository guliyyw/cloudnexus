/**
 * 跨模块跳转工具函数
 * 用于从文件预览、聊天等场景跳转到相册/音乐模块
 *
 * 注意：这些函数返回跳转参数，需在组件内配合 useNavigate 使用：
 *   const navigate = useNavigate()
 *   navigate(...getAlbumNavigateOpts(fileId))
 */

/** 构造跳转到相册的参数 */
export function getAlbumNavigateOpts(fileId: string) {
  return { pathname: '/album' as const, state: { highlightFileId: fileId } }
}

/** 构造跳转到音乐的参数 */
export function getMusicNavigateOpts(fileId: string, source: 'public' | 'cloud' = 'cloud') {
  return { pathname: '/music' as const, state: { playFileId: fileId, source } }
}

/** 判断 MIME 类型是否为图片 */
export function isImageMime(mime?: string): boolean {
  return !!mime?.startsWith('image/')
}

/** 判断 MIME 类型是否为音频 */
export function isAudioMime(mime?: string): boolean {
  return !!mime?.startsWith('audio/')
}
