package giphy

type (
	userValue struct {
		username string
		known    bool
	}

	userCache = lru[userValue]
)

func newUserCache(maxItems int) *userCache {
	return newLRU[userValue](maxItems)
}
