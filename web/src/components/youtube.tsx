export function YouTube({ id }: { id: string }) {
  return (
    <div className="relative my-6 aspect-video w-full overflow-hidden rounded-lg border border-neutral-800">
      <iframe
        src={`https://www.youtube.com/embed/${id}`}
        title="YouTube video"
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
        allowFullScreen
        className="absolute inset-0 h-full w-full"
      />
    </div>
  )
}
