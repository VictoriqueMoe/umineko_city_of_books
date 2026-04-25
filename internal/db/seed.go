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
	epitaphBody          = `<p><em>Now then, let us begin the game.</em></p>` +
		`<p>On the first twilight, the second bell tolls beneath the harbour.</p>` +
		`<p>On the second twilight, four candles gutter in the parlour.</p>` +
		`<p>On the third twilight, the eighth moth circles the lantern.</p>` +
		`<p>On the fourth twilight, the third blade rests upon the altar.</p>` +
		`<p>On the fifth twilight, the eleventh door refuses to open.</p>` +
		`<p>On the sixth twilight, seven seagulls wheel above the crest.</p>` +
		`<p>On the seventh twilight, the ninth chord resolves the hymn.</p>` +
		`<p>On the eighth twilight, the first rose bleeds upon the veranda.</p>` +
		`<p>On the ninth twilight, twelve bones lie unnamed beneath the garden.</p>` +
		`<p>On the tenth twilight, five witches bow to one another.</p>` +
		`<p>On the final twilight, ten coffins are sealed forever.</p>`
)

func SeedContent(db *sql.DB) error {
	if _, err := db.Exec(
		`INSERT INTO users (id, username, password_hash, display_name, bio)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO NOTHING`,
		MariaSeedUserID,
		"maria_u",
		mariaSeedPwdHash,
		"Maria Ushiromiya",
		"Uu~ mama said I shouldn't talk to strangers, so I left this here instead. If you find it, don't tell anyone.",
	); err != nil {
		return fmt.Errorf("seed maria user: %w", err)
	}

	if _, err := db.Exec(
		`INSERT INTO fanfics (id, user_id, title, summary, series, rating, language, status, is_oneshot)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (id) DO NOTHING`,
		EpitaphSeedFanficID,
		MariaSeedUserID,
		"The Witch's Epitaph",
		"A little something mama taught me. I think it's a riddle. Uu~",
		"Umineko",
		"K",
		"English",
		"complete",
		true,
	); err != nil {
		return fmt.Errorf("seed epitaph fanfic: %w", err)
	}

	if _, err := db.Exec(
		`INSERT INTO fanfic_chapters (id, fanfic_id, chapter_number, title, body, word_count)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO NOTHING`,
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
