package db

import (
	"database/sql"
	"fmt"
)

const (
	MariaSeedUserID      = "00000000-0000-4000-8000-000000000001"
	EpitaphSeedFanficID  = "00000000-0000-4000-8000-000000000002"
	epitaphSeedChapterID = "00000000-0000-4000-8000-000000000003"
	mariaSeedPwdHash     = "$2a$10$T0RCi7ybdIJraglqpfRiwOi4diB1y2/reIVa6YfWGtneJan4KwxLm"
	epitaphBody          = `Now then, let us begin the game.

On the first twilight, the second bell tolls beneath the harbour.

On the second twilight, four candles gutter in the parlour.

On the third twilight, the eighth moth circles the lantern.

On the fourth twilight, the third blade rests upon the altar.

On the fifth twilight, the eleventh door refuses to open.

On the sixth twilight, seven seagulls wheel above the crest.

On the seventh twilight, the ninth chord resolves the hymn.

On the eighth twilight, the first rose bleeds upon the veranda.

On the ninth twilight, twelve bones lie unnamed beneath the garden.

On the tenth twilight, five witches bow to one another.

On the final twilight, ten coffins are sealed forever.`
)

func SeedContent(db *sql.DB) error {
	if _, err := db.Exec(
		`INSERT OR IGNORE INTO users (id, username, password_hash, display_name, bio)
		 VALUES (?, ?, ?, ?, ?)`,
		MariaSeedUserID,
		"maria_u",
		mariaSeedPwdHash,
		"Maria Ushiromiya",
		"Uu~ mama said I shouldn't talk to strangers, so I left this here instead. If you find it, don't tell anyone.",
	); err != nil {
		return fmt.Errorf("seed maria user: %w", err)
	}

	if _, err := db.Exec(
		`INSERT OR IGNORE INTO fanfics (id, user_id, title, summary, series, rating, language, status, is_oneshot)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		EpitaphSeedFanficID,
		MariaSeedUserID,
		"The Witch's Epitaph",
		"A little something mama taught me. I think it's a riddle. Uu~",
		"Umineko",
		"K",
		"English",
		"complete",
		1,
	); err != nil {
		return fmt.Errorf("seed epitaph fanfic: %w", err)
	}

	if _, err := db.Exec(
		`INSERT OR IGNORE INTO fanfic_chapters (id, fanfic_id, chapter_number, title, body, word_count)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		epitaphSeedChapterID,
		EpitaphSeedFanficID,
		1,
		"The Epitaph",
		epitaphBody,
		80,
	); err != nil {
		return fmt.Errorf("seed epitaph chapter: %w", err)
	}

	return nil
}
