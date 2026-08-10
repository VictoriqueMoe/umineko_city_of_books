-- +goose Up
ALTER TABLE mysteries
    ADD COLUMN knox_culprit_named_early BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN knox_no_supernatural BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN knox_passages_declared BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN knox_no_unknown_poison BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN knox_no_outsider BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN knox_no_lucky_accident BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN knox_detective_not_culprit BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN knox_clues_shown BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN knox_narrator_hides_nothing BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN knox_no_unannounced_twins BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE mysteries
    DROP COLUMN knox_culprit_named_early,
    DROP COLUMN knox_no_supernatural,
    DROP COLUMN knox_passages_declared,
    DROP COLUMN knox_no_unknown_poison,
    DROP COLUMN knox_no_outsider,
    DROP COLUMN knox_no_lucky_accident,
    DROP COLUMN knox_detective_not_culprit,
    DROP COLUMN knox_clues_shown,
    DROP COLUMN knox_narrator_hides_nothing,
    DROP COLUMN knox_no_unannounced_twins;
