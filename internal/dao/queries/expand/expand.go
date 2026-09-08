package main

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type commentEntity struct {
	Entity     string
	Table      string
	FK         string
	LikesTable string
	MediaTable string
	KeyType    string
}

var commentEntities = []commentEntity{
	{Entity: "Post", Table: "post_comments", FK: "post_id", LikesTable: "post_comment_likes", MediaTable: "post_comment_media", KeyType: "uuid.UUID"},
	{Entity: "Art", Table: "art_comments", FK: "art_id", LikesTable: "art_comment_likes", MediaTable: "art_comment_media", KeyType: "uuid.UUID"},
	{Entity: "Announcement", Table: "announcement_comments", FK: "announcement_id", LikesTable: "announcement_comment_likes", MediaTable: "announcement_comment_media", KeyType: "uuid.UUID"},
	{Entity: "Mystery", Table: "mystery_comments", FK: "mystery_id", LikesTable: "mystery_comment_likes", MediaTable: "mystery_comment_media", KeyType: "uuid.UUID"},
	{Entity: "Ship", Table: "ship_comments", FK: "ship_id", LikesTable: "ship_comment_likes", MediaTable: "ship_comment_media", KeyType: "uuid.UUID"},
	{Entity: "OC", Table: "oc_comments", FK: "oc_id", LikesTable: "oc_comment_likes", MediaTable: "oc_comment_media", KeyType: "uuid.UUID"},
	{Entity: "Fanfic", Table: "fanfic_comments", FK: "fanfic_id", LikesTable: "fanfic_comment_likes", MediaTable: "fanfic_comment_media", KeyType: "uuid.UUID"},
	{Entity: "Journal", Table: "journal_comments", FK: "journal_id", LikesTable: "journal_comment_likes", MediaTable: "journal_comment_media", KeyType: "uuid.UUID"},
	{Entity: "Secret", Table: "secret_comments", FK: "secret_id", LikesTable: "secret_comment_likes", MediaTable: "secret_comment_media", KeyType: "string"},
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

	sqlTmpl, err := template.ParseFiles(filepath.Join(tmplDir, "comments.sql.tmpl"))
	if err != nil {
		fail(err)
	}

	for _, e := range commentEntities {
		var out strings.Builder
		if err := sqlTmpl.Execute(&out, e); err != nil {
			fail(err)
		}

		path := filepath.Join(genDir, e.Table+".sql")
		if err := os.WriteFile(path, []byte(out.String()), 0o644); err != nil {
			fail(err)
		}
	}

	bindTmpl, err := template.ParseFiles(filepath.Join(tmplDir, "comments_bind.go.tmpl"))
	if err != nil {
		fail(err)
	}

	type binding struct {
		commentEntity
		Recv    string
		FKField string
	}

	for _, e := range commentEntities {
		b := binding{
			commentEntity: e,
			Recv:          strings.ToLower(e.Entity[:1]) + e.Entity[1:] + "CommentQuerier",
			FKField:       fkField(e.FK),
		}

		var bind strings.Builder
		if err := bindTmpl.Execute(&bind, b); err != nil {
			fail(err)
		}

		formatted, err := format.Source([]byte(bind.String()))
		if err != nil {
			fail(fmt.Errorf("format %s: %w", e.Table, err))
		}

		bindPath := filepath.Join(root, "internal", "dao", e.Table+"_bind_gen.go")
		if err := os.WriteFile(bindPath, formatted, 0o644); err != nil {
			fail(err)
		}
	}

	fmt.Printf("expanded %d comment entities\n", len(commentEntities))
}

func fkField(fk string) string {
	parts := strings.Split(fk, "_")

	var out strings.Builder
	for _, p := range parts {
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
