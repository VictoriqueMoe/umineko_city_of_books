package giphy

type (
	cache = lru[*Response]
)

func newCache(maxItems int) *cache {
	return newLRU[*Response](maxItems)
}
