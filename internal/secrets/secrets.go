package secrets

type (
	ID string

	Spec struct {
		ID               ID
		ExpectedHash     string
		VanityRoleID     string
		Title            string
		Description      string
		Riddle           string
		Icon             string
		Pointer          string
		SolvedMessage    string
		ReadyPlaceholder string
		PendingHint      string
		ParentID         ID
		Pieces           []Piece
	}

	Piece struct {
		ID     ID
		Letter string
		Tile   int
	}
)

var registry = map[ID]Spec{}

func Register(specs ...Spec) {
	for i := 0; i < len(specs); i++ {
		registry[specs[i].ID] = specs[i]
	}
}

func Lookup(id string) (Spec, bool) {
	spec, ok := registry[ID(id)]
	return spec, ok
}

func All() []Spec {
	result := make([]Spec, 0, len(registry))
	for _, spec := range registry {
		result = append(result, spec)
	}
	return result
}

func WithVanityRole() []Spec {
	result := make([]Spec, 0, len(registry))
	for _, spec := range registry {
		if spec.VanityRoleID == "" {
			continue
		}
		result = append(result, spec)
	}
	return result
}

func Listed() []Spec {
	result := make([]Spec, 0, len(registry))
	for _, spec := range registry {
		if spec.Title == "" {
			continue
		}
		result = append(result, spec)
	}
	return result
}

func ParentOf(id ID) (Spec, bool) {
	spec, ok := registry[id]
	if !ok {
		return Spec{}, false
	}
	if spec.ParentID == "" {
		return spec, true
	}
	parent, ok := registry[spec.ParentID]
	return parent, ok
}

func PieceIDStrings(parent Spec) []string {
	result := make([]string, len(parent.Pieces))
	for i := 0; i < len(parent.Pieces); i++ {
		result[i] = string(parent.Pieces[i].ID)
	}
	return result
}
