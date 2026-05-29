import { authorizedFetch } from './api'

export async function transcribeAudioFallback(audio: Blob, filename = 'audio.webm'): Promise<string> {
  const form = new FormData()
  form.append('audio', audio, filename)
  const res = await authorizedFetch('/api/v1/stt/transcribe', {
    method: 'POST',
    body: form,
  })
  if (!res.ok) {
    throw new Error('Server transcription failed')
  }
  const data = (await res.json()) as { transcript?: string }
  return data.transcript ?? ''
}
