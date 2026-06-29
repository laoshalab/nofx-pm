type Props = {
  hasMore: boolean
  loading: boolean
  onLoadMore: () => void
  label?: string
}

export function PaginationBar({ hasMore, loading, onLoadMore, label = '加载更多' }: Props) {
  if (!hasMore) return null
  return (
    <div className="pt-3 flex justify-center">
      <button
        type="button"
        disabled={loading}
        onClick={onLoadMore}
        className="text-xs px-3 py-1.5 rounded bg-zinc-800 text-zinc-300 hover:bg-zinc-700 disabled:opacity-50"
      >
        {loading ? '加载中…' : label}
      </button>
    </div>
  )
}
