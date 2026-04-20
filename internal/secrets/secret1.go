package secrets

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

var witchHunterPieces = []Piece{
	{ID: Piece01, Letter: "R", Tile: 1},
	{ID: Piece02, Letter: "M", Tile: 2},
	{ID: Piece03, Letter: "I", Tile: 3},
	{ID: Piece04, Letter: "A", Tile: 4},
	{ID: Piece05, Letter: "A", Tile: 5},
	{ID: Piece06, Letter: "K", Tile: 6},
	{ID: Piece07, Letter: "I", Tile: 7},
	{ID: Piece08, Letter: "G", Tile: 8},
	{ID: Piece09, Letter: "S", Tile: 9},
	{ID: Piece10, Letter: "L", Tile: 10},
	{ID: Piece11, Letter: "C", Tile: 11},
	{ID: Piece12, Letter: "E", Tile: 12},
}

var witchHunterRiddle = `Somewhere on this site, Maria Ushiromiya has scattered twelve pieces of an old epitaph.

Each piece is a single letter hiding somewhere ordinary, tucked beside a tagline, a button, a rule, a subtitle, a chip, a sentence you might glance past. Find them by looking for the little sparkle she left behind, and your profile will remember them for you. Mama is tricky; they are not always in the same kind of place.

When you have gathered all twelve, read mama's writing carefully. Her words do not spell the answer; they point at it. Follow her count and the answer will speak itself.

Whisper that word back to her from your own profile panel. If you are right, the Endless Witch will teach you her colours, and everyone watching this page will know who spoke first.

Mama wrote me a little rhyme to help me remember where I hid each one. She said I could share it, but only if I asked nicely. Uu~.

Uu~ Mama's Twelve Little Places

One, on the page that greets you first, tucked beside the witch's famous little line.
Two, down at the very bottom of the garden, next to the hand that made it with love.
Three, where voices gather in parlours, hiding on the button that opens a new door.
Four, where the top sleuths of the city are ranked, right beside the heading that names them.
Five, where witches trade their theories in red and blue and gold and purple, perched on the button that measures our trust.
Six, where lovers are tied with a ribbon, a little herring hides in the rules that govern them (mii~).
Seven, where stories are dreamt again, tucked into the welcome the archive whispers.
Eight, where readers keep all their evenings, resting on the button that starts a new one.
Nine, where pictures hang in silver frames, at the end of the paragraph that tells you how it works.
Ten, where other witches speak their lines, watching quietly from a corner of the page.
Eleven, where a page is asked for but not found, hidden in the middle of the witch's shrug.
Twelve, where every player of this city is listed, standing beside the ones who are online.

Uu~ that's all. One of the twelve is only a mask; mama always tests me like that.`

func init() {
	Register(
		Spec{
			ID:               WitchHunter,
			ExpectedHash:     "31bae96b737325e614a6fd336c4bb00948db442184a7671a6d6aaff52163eb04",
			VanityRoleID:     "system_witch_hunter",
			Title:            "The Witch's Epitaph",
			Description:      "Maria scattered an old epitaph across the site. Collect the twelve pieces, decode mama's riddle, and speak the witch's name.",
			Riddle:           witchHunterRiddle,
			Icon:             "\u273F",
			Pointer:          "Mama wrote something somewhere on this site. Find her words, and you'll know which letters to speak.",
			SolvedMessage:    "Uu~ the Endless Witch has taught you her secret. The Maria theme and the Witch Hunter role are yours.",
			ReadyPlaceholder: "Whisper mama's truth...",
			PendingHint:      "Find all twelve pieces before the witch will hear you.",
			Pieces:           witchHunterPieces,
		},
		Spec{ID: Piece01, ExpectedHash: "316ead35863841be0200c65fd455c33c1b6da7fa81874ea818e428ebe06d6152", ParentID: WitchHunter},
		Spec{ID: Piece02, ExpectedHash: "a5d655eaa8ee17314974bad4673cbf6f70ffb2ca74a68bf5462ba7aeacd5b801", ParentID: WitchHunter},
		Spec{ID: Piece03, ExpectedHash: "edafd7c0b2496569043d3a79634a6d5628adba0088823a8113a16f51aefdeaa7", ParentID: WitchHunter},
		Spec{ID: Piece04, ExpectedHash: "7180436b142fcd333ab8ad326b880d818e428f4007a3aefa639505bbb9b9a032", ParentID: WitchHunter},
		Spec{ID: Piece05, ExpectedHash: "a85f6b627a1ba9d0d6380c91c96a3f7ecb3ddfb5889d4fe692363d2e20b03da2", ParentID: WitchHunter},
		Spec{ID: Piece06, ExpectedHash: "fa7b51d6fb4016299dccd5493ef3c2ff7c08fb8fc44f9456e581740dcd0b1fc3", ParentID: WitchHunter},
		Spec{ID: Piece07, ExpectedHash: "47d9dc6dae2f8bd83f779be01e93ab0dec7dc069ba0b23c8a118b650b119fea2", ParentID: WitchHunter},
		Spec{ID: Piece08, ExpectedHash: "019926bb0ea43a839b164c658264e48f078f1379083b3b721e7c1b7903762cbe", ParentID: WitchHunter},
		Spec{ID: Piece09, ExpectedHash: "43a1f09c93fa4571e86abb7b22560ce2c167cd01955586366277a1cbafa9280c", ParentID: WitchHunter},
		Spec{ID: Piece10, ExpectedHash: "851e817b8858e15d7941175d74fc66fdda1362326a65057212889f1a933891af", ParentID: WitchHunter},
		Spec{ID: Piece11, ExpectedHash: "56980949c74305bee43edbaacbb85d1a2e0d9b18339a48e45f3323e10c983ad2", ParentID: WitchHunter},
		Spec{ID: Piece12, ExpectedHash: "6f0d6c6f23fb58baccab878504508ceeceef0df91c59d3b69cea49b01f3da519", ParentID: WitchHunter},
	)
}
