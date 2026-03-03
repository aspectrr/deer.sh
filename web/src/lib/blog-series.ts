export interface SeriesChapter {
  part: number
  slug: string
  shortTitle: string
}

export interface BlogSeries {
  id: string
  title: string
  subtitle: string
  description: string
  slugPrefix: string
  author: string
  authorImage: string
  chapters: SeriesChapter[]
}

export const blogSeries: BlogSeries[] = [
  {
    id: 'hypervisor',
    title: 'Building a Hypervisor from Scratch',
    subtitle: 'Apple Silicon, Rust, zero dependencies',
    description:
      'A 10-part series on writing a type-1 hypervisor targeting Apple Silicon using Rust and the Hypervisor.framework API. From raw FFI bindings to booting Linux with an interactive shell.',
    slugPrefix: 'hypervisor-pt',
    author: 'Collin @ Fluid.sh',
    authorImage: '/images/skeleton_smoking_cigarette.jpg',
    chapters: [
      {
        part: 1,
        slug: 'hypervisor-pt1-what-happens',
        shortTitle: 'What Happens When You Run a VM?',
      },
      { part: 2, slug: 'hypervisor-pt2-raw-metal', shortTitle: 'Raw Metal: FFI Bindings' },
      { part: 3, slug: 'hypervisor-pt3-safe-abstractions', shortTitle: 'Safe Abstractions' },
      { part: 4, slug: 'hypervisor-pt4-exit-loop', shortTitle: 'The Exit Loop' },
      { part: 5, slug: 'hypervisor-pt5-mmio', shortTitle: 'MMIO: Faking Hardware' },
      { part: 6, slug: 'hypervisor-pt6-serial-console', shortTitle: 'Serial Console: PL011 UART' },
      { part: 7, slug: 'hypervisor-pt7-loading-binaries', shortTitle: 'Loading Real Binaries' },
      { part: 8, slug: 'hypervisor-pt8-device-tree', shortTitle: 'The Device Tree' },
      { part: 9, slug: 'hypervisor-pt9-booting-linux', shortTitle: 'Booting Linux' },
      { part: 10, slug: 'hypervisor-pt10-complete-machine', shortTitle: 'The Complete Machine' },
    ],
  },
]

export function findSeriesForSlug(
  slug: string
): { series: BlogSeries; chapterIndex: number } | null {
  for (const series of blogSeries) {
    const idx = series.chapters.findIndex((ch) => ch.slug === slug)
    if (idx !== -1) return { series, chapterIndex: idx }
  }
  return null
}

const _seriesSlugs = new Set(blogSeries.flatMap((s) => s.chapters.map((ch) => ch.slug)))

export function getSeriesSlugs(): Set<string> {
  return _seriesSlugs
}
