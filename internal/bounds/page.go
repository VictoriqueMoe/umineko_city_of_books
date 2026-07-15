package bounds

const (
	DefaultLimit = 20
	MaxLimit     = 100
	MaxOffset    = 1000
)

type (
	Page struct {
		limit  int
		offset int
	}
)

func NewPage(limit, offset int) Page {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	if offset < 0 {
		offset = 0
	}
	if offset > MaxOffset {
		offset = MaxOffset
	}

	return Page{limit: limit, offset: offset}
}

func (p Page) Limit() int {
	if p.limit <= 0 {
		return DefaultLimit
	}
	if p.limit > MaxLimit {
		return MaxLimit
	}

	return p.limit
}

func (p Page) Offset() int {
	if p.offset < 0 {
		return 0
	}
	if p.offset > MaxOffset {
		return MaxOffset
	}

	return p.offset
}

func (p Page) Window() int {
	return p.Limit() + p.Offset()
}
