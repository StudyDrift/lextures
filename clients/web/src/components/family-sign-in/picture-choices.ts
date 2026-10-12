/** Picture-password alphabet (#771). Must match server `managedlearners.PictureChoices`. */
export const PICTURE_CHOICES = ['apple', 'ball', 'cat', 'dog', 'fish', 'moon', 'star', 'sun', 'tree'] as const

export const PICTURE_MIN = 3
export const PICTURE_MAX = 6
export const PIN_MIN = 4
export const PIN_MAX = 6

const PICTURES: Record<string, { emoji: string; label: string }> = {
  apple: { emoji: '🍎', label: 'Apple' },
  ball: { emoji: '⚽', label: 'Ball' },
  cat: { emoji: '🐱', label: 'Cat' },
  dog: { emoji: '🐶', label: 'Dog' },
  fish: { emoji: '🐟', label: 'Fish' },
  moon: { emoji: '🌙', label: 'Moon' },
  star: { emoji: '⭐', label: 'Star' },
  sun: { emoji: '☀️', label: 'Sun' },
  tree: { emoji: '🌳', label: 'Tree' },
}

export function pictureFor(id: string): { emoji: string; label: string } {
  return PICTURES[id] ?? { emoji: '❔', label: id }
}

export function isValidPin(pin: string): boolean {
  return new RegExp(`^[0-9]{${PIN_MIN},${PIN_MAX}}$`).test(pin)
}
