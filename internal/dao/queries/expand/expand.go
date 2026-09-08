package main

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type (
	entity struct {
		Entity     string
		Table      string
		FK         string
		LikesTable string
		MediaTable string
		KeyType    string
	}

	family struct {
		Name     string
		Template string
		Suffix   string
		Entities []entity
	}

	binding struct {
		entity
		Recv    string
		FKField string
	}
)

var families = []family{
	{
		Name:     "comments",
		Template: "comments",
		Suffix:   "Comment",
		Entities: []entity{
			{Entity: "Post", Table: "post_comments", FK: "post_id", LikesTable: "post_comment_likes", MediaTable: "post_comment_media", KeyType: "uuid.UUID"},
			{Entity: "Art", Table: "art_comments", FK: "art_id", LikesTable: "art_comment_likes", MediaTable: "art_comment_media", KeyType: "uuid.UUID"},
			{Entity: "Announcement", Table: "announcement_comments", FK: "announcement_id", LikesTable: "announcement_comment_likes", MediaTable: "announcement_comment_media", KeyType: "uuid.UUID"},
			{Entity: "Mystery", Table: "mystery_comments", FK: "mystery_id", LikesTable: "mystery_comment_likes", MediaTable: "mystery_comment_media", KeyType: "uuid.UUID"},
			{Entity: "Ship", Table: "ship_comments", FK: "ship_id", LikesTable: "ship_comment_likes", MediaTable: "ship_comment_media", KeyType: "uuid.UUID"},
			{Entity: "OC", Table: "oc_comments", FK: "oc_id", LikesTable: "oc_comment_likes", MediaTable: "oc_comment_media", KeyType: "uuid.UUID"},
			{Entity: "Fanfic", Table: "fanfic_comments", FK: "fanfic_id", LikesTable: "fanfic_comment_likes", MediaTable: "fanfic_comment_media", KeyType: "uuid.UUID"},
			{Entity: "Journal", Table: "journal_comments", FK: "journal_id", LikesTable: "journal_comment_likes", MediaTable: "journal_comment_media", KeyType: "uuid.UUID"},
			{Entity: "Secret", Table: "secret_comments", FK: "secret_id", LikesTable: "secret_comment_likes", MediaTable: "secret_comment_media", KeyType: "string"},
		},
	},
	{
		Name:     "likes",
		Template: "likes",
		Suffix:   "Like",
		Entities: []entity{
			{Entity: "Post", Table: "post_likes", FK: "post_id"},
			{Entity: "Art", Table: "art_likes", FK: "art_id"},
		},
	},
	{
		Name:     "views",
		Template: "views",
		Suffix:   "View",
		Entities: []entity{
			{Entity: "Post", Table: "post_views", FK: "post_id"},
			{Entity: "Art", Table: "art_views", FK: "art_id"},
			{Entity: "Fanfic", Table: "fanfic_views", FK: "fanfic_id"},
		},
	},
	{
		Name:     "votes",
		Template: "votes",
		Suffix:   "Vote",
		Entities: []entity{
			{Entity: "Theory", Table: "theory_votes", FK: "theory_id"},
			{Entity: "Response", Table: "response_votes", FK: "response_id"},
			{Entity: "MysteryAttempt", Table: "mystery_attempt_votes", FK: "attempt_id"},
			{Entity: "Ship", Table: "ship_votes", FK: "ship_id"},
			{Entity: "OC", Table: "oc_votes", FK: "oc_id"},
		},
	},
	{
		Name:     "owned",
		Template: "owned",
		Suffix:   "Owned",
		Entities: []entity{
			{Entity: "Post", Table: "posts"},
			{Entity: "Art", Table: "art"},
			{Entity: "Mystery", Table: "mysteries"},
			{Entity: "Ship", Table: "ships"},
			{Entity: "OC", Table: "ocs"},
			{Entity: "Fanfic", Table: "fanfics"},
			{Entity: "Journal", Table: "journals"},
		},
	},
	{
		Name:     "media",
		Template: "media",
		Suffix:   "Media",
		Entities: []entity{
			{Entity: "Post", Table: "post_media", FK: "post_id"},
			{Entity: "Mystery", Table: "mystery_media", FK: "mystery_id"},
			{Entity: "JournalEntry", Table: "journal_entry_media", FK: "entry_id"},
		},
	},
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}

	tmplDir := filepath.Join(root, "internal", "dao", "queries", "templates")
	genDir := filepath.Join(root, "internal", "dao", "queries", "gen")

	if err := os.RemoveAll(genDir); err != nil {
		fail(err)
	}

	if err := os.MkdirAll(genDir, 0o755); err != nil {
		fail(err)
	}

	for _, old := range mustGlob(filepath.Join(root, "internal", "dao", "*_bind_gen.go")) {
		if err := os.Remove(old); err != nil {
			fail(err)
		}
	}

	total := 0
	for _, f := range families {
		sqlPath := filepath.Join(tmplDir, f.Template+".sql.tmpl")
		sqlTmpl, err := template.ParseFiles(sqlPath)
		if err != nil {
			fail(err)
		}

		bindPath := filepath.Join(tmplDir, f.Template+"_bind.go.tmpl")
		bindTmpl, bindErr := template.ParseFiles(bindPath)

		for _, e := range f.Entities {
			writeRendered(sqlTmpl, e, filepath.Join(genDir, f.Name+"_"+e.Table+".sql"), false)

			if bindErr != nil {
				continue
			}

			b := binding{
				entity:  e,
				Recv:    lowerFirst(e.Entity) + f.Suffix + "Querier",
				FKField: fkField(e.FK),
			}

			writeRendered(bindTmpl, b, filepath.Join(root, "internal", "dao", f.Name+"_"+e.Table+"_bind_gen.go"), true)
			total++
		}
	}

	fmt.Printf("expanded %d families, %d bindings\n", len(families), total)
}

func writeRendered(t *template.Template, data any, path string, gofmtIt bool) {
	var out strings.Builder
	if err := t.Execute(&out, data); err != nil {
		fail(err)
	}

	body := []byte(out.String())
	if gofmtIt {
		formatted, err := format.Source(body)
		if err != nil {
			fail(fmt.Errorf("format %s: %w", path, err))
		}

		body = formatted
	}

	if err := os.WriteFile(path, body, 0o644); err != nil {
		fail(err)
	}
}

func mustGlob(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fail(err)
	}

	return matches
}

func lowerFirst(s string) string {
	return strings.ToLower(s[:1]) + s[1:]
}

func fkField(fk string) string {
	if fk == "" {
		return ""
	}

	var out strings.Builder
	for p := range strings.SplitSeq(fk, "_") {
		switch p {
		case "id":
			out.WriteString("ID")
		case "oc":
			out.WriteString("Oc")
		default:
			out.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}

	return out.String()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "expand:", err)
	os.Exit(1)
}
