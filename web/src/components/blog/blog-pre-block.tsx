import { type ReactNode, isValidElement, Children } from 'react'
import { IdeCodeBlock } from './ide-code-block'
import { TerminalCodeBlock } from './terminal-code-block'

interface BlogPreBlockProps extends React.HTMLAttributes<HTMLPreElement> {
  children?: ReactNode
}

export function BlogPreBlock(props: BlogPreBlockProps) {
  const codeChild = findCodeChild(props.children)
  if (!codeChild) {
    return <pre {...props} />
  }

  const className = (codeChild.props.className as string) || ''
  const lang = className.replace(/^language-/, '') || undefined
  const rawText = extractText(codeChild.props.children)

  if (lang === 'terminal') {
    const { title, code } = extractTerminalTitle(rawText)
    return <TerminalCodeBlock code={code} title={title} />
  }

  const { filename, code } = extractFilename(rawText, lang)
  return <IdeCodeBlock code={code} lang={lang} filename={filename} />
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function findCodeChild(children: ReactNode): React.ReactElement<any> | null {
  const arr = Children.toArray(children)
  for (const child of arr) {
    if (
      isValidElement(child) &&
      (child.type === 'code' || (child.props as { mdxType?: string }).mdxType === 'code')
    ) {
      return child
    }
  }
  return null
}

function extractText(node: ReactNode): string {
  if (typeof node === 'string') return node
  if (typeof node === 'number') return String(node)
  if (!node) return ''
  if (Array.isArray(node)) return node.map(extractText).join('')
  if (isValidElement(node)) {
    return extractText((node.props as { children?: ReactNode }).children)
  }
  return ''
}

const FILENAME_PATTERNS: Record<string, RegExp> = {
  slash: /^\/\/\s+(\S+\.\S+)\s*$/,
  hash: /^#\s+(\S+\.\S+)\s*$/,
  xml: /^<!--\s+(\S+\.\S+)\s+-->\s*$/,
}

function extractFilename(
  raw: string,
  lang?: string
): { filename: string | undefined; code: string } {
  const lines = raw.split('\n')
  if (lines.length < 2) return { filename: undefined, code: raw }

  const first = lines[0]

  // // path/to/file.ext - for rust, go, c, typescript, javascript, etc.
  const slashMatch = first.match(FILENAME_PATTERNS.slash)
  if (slashMatch) {
    return { filename: slashMatch[1], code: lines.slice(1).join('\n') }
  }

  // # path/to/file.ext - for bash, yaml, toml, python - but NOT shebangs
  if (lang !== 'bash' || !first.startsWith('#!')) {
    const hashMatch = first.match(FILENAME_PATTERNS.hash)
    if (hashMatch) {
      return { filename: hashMatch[1], code: lines.slice(1).join('\n') }
    }
  }

  // <!-- path/to/file.ext --> - for xml, html
  const xmlMatch = first.match(FILENAME_PATTERNS.xml)
  if (xmlMatch) {
    return { filename: xmlMatch[1], code: lines.slice(1).join('\n') }
  }

  return { filename: undefined, code: raw }
}

function extractTerminalTitle(raw: string): { title: string | undefined; code: string } {
  const lines = raw.split('\n')
  if (lines.length < 2) return { title: undefined, code: raw }

  // # title line at the top (comment-style title)
  const first = lines[0]
  const match = first.match(/^#\s+(.+)\s*$/)
  if (match && !first.startsWith('#!')) {
    return { title: match[1], code: lines.slice(1).join('\n') }
  }

  return { title: undefined, code: raw }
}
