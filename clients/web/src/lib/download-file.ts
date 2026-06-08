import { authorizedFetch } from './api'

export async function downloadAuthorizedFile(filePath: string, filename: string): Promise<void> {
  const res = await authorizedFetch(filePath)
  if (!res.ok) throw new Error('Download failed.')
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
