import type React from 'react'
import type { SyntaxHighlighterProps } from 'react-syntax-highlighter'

declare module 'react-syntax-highlighter/dist/esm/prism-light' {
  class PrismLight extends React.Component<SyntaxHighlighterProps> {
    static registerLanguage(name: string, func: unknown): void
  }
  export = PrismLight
}

type PrismStyle = Record<string, React.CSSProperties>

declare module 'react-syntax-highlighter/dist/esm/styles/prism' {
  export const a11yDark: PrismStyle
  export const atomDark: PrismStyle
  export const base16AteliersulphurpoolLight: PrismStyle
  export const cb: PrismStyle
  export const coldarkCold: PrismStyle
  export const coldarkDark: PrismStyle
  export const coyWithoutShadows: PrismStyle
  export const darcula: PrismStyle
  export const dracula: PrismStyle
  export const duotoneDark: PrismStyle
  export const duotoneLight: PrismStyle
  export const ghcolors: PrismStyle
  export const gruvboxDark: PrismStyle
  export const gruvboxLight: PrismStyle
  export const holiTheme: PrismStyle
  export const hopscotch: PrismStyle
  export const lucario: PrismStyle
  export const materialDark: PrismStyle
  export const materialLight: PrismStyle
  export const materialOceanic: PrismStyle
  export const nightOwl: PrismStyle
  export const nord: PrismStyle
  export const okaidia: PrismStyle
  export const oneDark: PrismStyle
  export const oneLight: PrismStyle
  export const pojoaque: PrismStyle
  export const prism: PrismStyle
  export const shadesOfPurple: PrismStyle
  export const solarizedlight: PrismStyle
  export const synthwave84: PrismStyle
  export const tomorrow: PrismStyle
  export const twilight: PrismStyle
  export const vs: PrismStyle
  export const vscDarkPlus: PrismStyle
  export const xonokai: PrismStyle
  export const zTouch: PrismStyle
}

declare module 'react-syntax-highlighter/dist/esm/languages/prism/*' {
  const language: unknown
  export default language
}
