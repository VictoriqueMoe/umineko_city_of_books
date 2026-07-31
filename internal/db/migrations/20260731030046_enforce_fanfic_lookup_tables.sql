-- +goose Up

UPDATE fanfics SET series = 'Umineko' WHERE btrim(series) = '';
UPDATE fanfics SET language = 'English' WHERE btrim(language) = '';

ALTER TABLE fanfics ALTER COLUMN series TYPE CITEXT USING series::citext;
ALTER TABLE fanfics ALTER COLUMN language TYPE CITEXT USING language::citext;

INSERT INTO fanfic_series (name) SELECT DISTINCT series FROM fanfics ON CONFLICT DO NOTHING;
INSERT INTO fanfic_languages (name) SELECT DISTINCT language FROM fanfics ON CONFLICT DO NOTHING;

ALTER TABLE fanfics ADD CONSTRAINT fanfics_series_fkey
    FOREIGN KEY (series) REFERENCES fanfic_series (name) ON UPDATE CASCADE;
ALTER TABLE fanfics ADD CONSTRAINT fanfics_language_fkey
    FOREIGN KEY (language) REFERENCES fanfic_languages (name) ON UPDATE CASCADE;

-- +goose Down

ALTER TABLE fanfics DROP CONSTRAINT fanfics_language_fkey;
ALTER TABLE fanfics DROP CONSTRAINT fanfics_series_fkey;

ALTER TABLE fanfics ALTER COLUMN language TYPE TEXT USING language::text;
ALTER TABLE fanfics ALTER COLUMN series TYPE TEXT USING series::text;
