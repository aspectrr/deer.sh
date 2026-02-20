import { type HighlighterCore, createHighlighterCore } from 'shiki/core'
import { createOnigurumaEngine } from 'shiki/engine/oniguruma'

let highlighter: HighlighterCore | null = null
let loading: Promise<HighlighterCore> | null = null

export async function getHighlighter(): Promise<HighlighterCore> {
  if (highlighter) return highlighter
  if (loading) return loading

  loading = createHighlighterCore({
    themes: [import('shiki/themes/github-dark.mjs')],
    langs: [
      import('shiki/langs/bash.mjs'),
      import('shiki/langs/yaml.mjs'),
      import('shiki/langs/json.mjs'),
      import('shiki/langs/go.mjs'),
      import('shiki/langs/typescript.mjs'),
      import('shiki/langs/proto.mjs'),
    ],
    engine: createOnigurumaEngine(import('shiki/wasm')),
  })
    .then((h) => {
      highlighter = h
      return h
    })
    .catch((err) => {
      // Reset so subsequent calls retry initialization
      loading = null
      throw err
    })

  return loading
}
