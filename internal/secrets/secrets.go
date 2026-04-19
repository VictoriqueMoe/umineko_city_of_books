package secrets

type (
	ID string

	Spec struct {
		ID           ID
		ExpectedHash string
		VanityRoleID string
		Title        string
		Description  string
		Riddle       string
		ParentID     ID
		PieceIDs     []ID
	}
)

const (
	WitchHunter ID = "witchHunter"
	Piece01     ID = "piece_01"
	Piece02     ID = "piece_02"
	Piece03     ID = "piece_03"
	Piece04     ID = "piece_04"
	Piece05     ID = "piece_05"
	Piece06     ID = "piece_06"
	Piece07     ID = "piece_07"
	Piece08     ID = "piece_08"
	Piece09     ID = "piece_09"
	Piece10     ID = "piece_10"
	Piece11     ID = "piece_11"
	Piece12     ID = "piece_12"
)

var witchHunterPieces = []ID{
	Piece01, Piece02, Piece03, Piece04, Piece05, Piece06,
	Piece07, Piece08, Piece09, Piece10, Piece11, Piece12,
}

var witchHunterRiddle = `Somewhere on this site, Maria Ushiromiya has scattered twelve pieces of an old epitaph.

Each piece is a single letter hiding next to an everyday heading in the garden of this city. Find them by looking for the little sparkle she left behind, and your profile will remember them for you.

When you have gathered all twelve, read mama's writing carefully. Her words do not spell the answer; they point at it. Follow her count and the answer will speak itself.

Whisper that word back to her from your own profile panel. If you are right, the Endless Witch will teach you her colours, and everyone watching this page will know who spoke first.`

var registry = map[ID]Spec{
	WitchHunter: {
		ID:           WitchHunter,
		ExpectedHash: "31bae96b737325e614a6fd336c4bb00948db442184a7671a6d6aaff52163eb04",
		VanityRoleID: "system_witch_hunter",
		Title:        "The Witch's Epitaph",
		Description:  "Maria scattered an old epitaph across the site. Collect the twelve pieces, decode mama's riddle, and speak the witch's name.",
		Riddle:       witchHunterRiddle,
		PieceIDs:     witchHunterPieces,
	},
	Piece01: {ID: Piece01, ExpectedHash: "316ead35863841be0200c65fd455c33c1b6da7fa81874ea818e428ebe06d6152", ParentID: WitchHunter},
	Piece02: {ID: Piece02, ExpectedHash: "a5d655eaa8ee17314974bad4673cbf6f70ffb2ca74a68bf5462ba7aeacd5b801", ParentID: WitchHunter},
	Piece03: {ID: Piece03, ExpectedHash: "edafd7c0b2496569043d3a79634a6d5628adba0088823a8113a16f51aefdeaa7", ParentID: WitchHunter},
	Piece04: {ID: Piece04, ExpectedHash: "7180436b142fcd333ab8ad326b880d818e428f4007a3aefa639505bbb9b9a032", ParentID: WitchHunter},
	Piece05: {ID: Piece05, ExpectedHash: "a85f6b627a1ba9d0d6380c91c96a3f7ecb3ddfb5889d4fe692363d2e20b03da2", ParentID: WitchHunter},
	Piece06: {ID: Piece06, ExpectedHash: "fa7b51d6fb4016299dccd5493ef3c2ff7c08fb8fc44f9456e581740dcd0b1fc3", ParentID: WitchHunter},
	Piece07: {ID: Piece07, ExpectedHash: "47d9dc6dae2f8bd83f779be01e93ab0dec7dc069ba0b23c8a118b650b119fea2", ParentID: WitchHunter},
	Piece08: {ID: Piece08, ExpectedHash: "019926bb0ea43a839b164c658264e48f078f1379083b3b721e7c1b7903762cbe", ParentID: WitchHunter},
	Piece09: {ID: Piece09, ExpectedHash: "43a1f09c93fa4571e86abb7b22560ce2c167cd01955586366277a1cbafa9280c", ParentID: WitchHunter},
	Piece10: {ID: Piece10, ExpectedHash: "851e817b8858e15d7941175d74fc66fdda1362326a65057212889f1a933891af", ParentID: WitchHunter},
	Piece11: {ID: Piece11, ExpectedHash: "56980949c74305bee43edbaacbb85d1a2e0d9b18339a48e45f3323e10c983ad2", ParentID: WitchHunter},
	Piece12: {ID: Piece12, ExpectedHash: "6f0d6c6f23fb58baccab878504508ceeceef0df91c59d3b69cea49b01f3da519", ParentID: WitchHunter},
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
	result := make([]string, len(parent.PieceIDs))
	for i := 0; i < len(parent.PieceIDs); i++ {
		result[i] = string(parent.PieceIDs[i])
	}
	return result
}
