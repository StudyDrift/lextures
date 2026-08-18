import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ArticleSocialImageField } from '../article-social-image-field'

const uploadMarketingMedia = vi.fn()
const getMarketingMedia = vi.fn()
vi.mock('../../../../lib/marketing-content-media-api', () => ({
  uploadMarketingMedia: (...args: unknown[]) => uploadMarketingMedia(...args),
  getMarketingMedia: (...args: unknown[]) => getMarketingMedia(...args),
  marketingMediaPreviewUrl: (asset: { renditions?: Array<{ name: string; url: string }> }) =>
    asset.renditions?.find((item) => item.name === 'social')?.url ?? asset.renditions?.[0]?.url ?? null,
}))

vi.mock('../../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ ffMotionControls: false }),
}))

describe('ArticleSocialImageField', () => {
  beforeEach(() => {
    uploadMarketingMedia.mockReset()
    getMarketingMedia.mockReset()
  })

  it('uploads an image and stores the media id', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    uploadMarketingMedia.mockResolvedValue({
      id: 'media-1',
      renditions: [{ name: 'social', ext: 'png', url: '/api/v1/public/content/media/media-1/social.png' }],
    })

    render(<ArticleSocialImageField title="Advice" onChange={onChange} />)
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['png'], 'card.png', { type: 'image/png' })
    await user.upload(input, file)

    expect(uploadMarketingMedia).toHaveBeenCalledWith(file, {
      altText: 'Advice',
      title: 'Advice',
    })
    expect(onChange).toHaveBeenCalledWith('media-1')
    expect(screen.getByRole('button', { name: 'Remove image' })).toBeInTheDocument()
  })
})
