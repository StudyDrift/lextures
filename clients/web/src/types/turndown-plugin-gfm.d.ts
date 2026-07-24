declare module 'turndown-plugin-gfm' {
  import type TurndownService from 'turndown'

  type Plugin = TurndownService.Plugin

  export const gfm: Plugin
  export const tables: Plugin
  export const strikethrough: Plugin
  export const taskListItems: Plugin
  export const highlightedCodeBlock: Plugin
}
