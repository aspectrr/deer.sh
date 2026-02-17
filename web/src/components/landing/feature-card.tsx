export function FeatureCard({
  iconString,
  title,
  description,
}: {
  iconString: string
  title: string
  description: string
}) {
  return (
    <div className="group rounded-lg border border-neutral-800 bg-neutral-900/50 p-4 transition-all duration-300 hover:border-blue-500/30">
      <div className="flex items-start gap-3">
        <div className="font-mono text-lg/snug text-blue-400">{iconString}</div>
        <div>
          <h3 className="mb-1 font-medium text-neutral-200">{title}</h3>
          <p className="text-sm text-neutral-500">{description}</p>
        </div>
      </div>
    </div>
  )
}
