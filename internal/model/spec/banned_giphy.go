package spec

type (
	NewBannedGiphy struct {
		Kind      string
		Value     string
		Reason    string
		CreatedBy *string
	}

	BannedGiphyDeletion struct {
		Kind  string
		Value string
	}
)
