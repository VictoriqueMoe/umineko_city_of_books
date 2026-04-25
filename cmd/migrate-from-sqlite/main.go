package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type (
	columnInfo struct {
		Name     string
		DataType string
		Nullable bool
	}

	tableInfo struct {
		Name           string
		Columns        []columnInfo
		HasIdentityCol bool
	}

	migrator struct {
		src       *sql.DB
		dst       *sql.DB
		tables    []tableInfo
		srcCounts map[string]int
		verbose   bool
	}
)

var (
	timestampFormats = []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
)

func main() {
	var (
		sqlitePath = flag.String("sqlite", "data/truths.db", "path to source SQLite database")
		pgDSN      = flag.String("pg", "", "target Postgres DSN (postgres://user:pass@host:port/db?sslmode=disable)")
		verbose    = flag.Bool("v", false, "verbose per-table progress")
	)
	flag.Parse()

	if *pgDSN == "" {
		fmt.Fprintln(os.Stderr, "missing -pg DSN")
		os.Exit(2)
	}

	src, err := sql.Open("sqlite", "file:"+*sqlitePath+"?mode=ro")
	if err != nil {
		fail("open sqlite", err)
	}
	defer src.Close()
	if err := src.Ping(); err != nil {
		fail("ping sqlite", err)
	}

	pgCfg, err := pgx.ParseConfig(*pgDSN)
	if err != nil {
		fail("parse pg dsn", err)
	}
	dst := stdlib.OpenDB(*pgCfg)
	defer dst.Close()
	if err := dst.Ping(); err != nil {
		fail("ping postgres", err)
	}

	m := &migrator{src: src, dst: dst, srcCounts: map[string]int{}, verbose: *verbose}

	fmt.Println("loading schema metadata from postgres...")
	if err := m.loadTargetSchema(); err != nil {
		fail("load schema", err)
	}
	fmt.Printf("found %d tables to migrate\n", len(m.tables))

	fmt.Println("counting source rows...")
	if err := m.countSourceRows(); err != nil {
		fail("count source", err)
	}

	fmt.Println("starting load transaction...")
	ctx := context.Background()
	tx, err := dst.BeginTx(ctx, nil)
	if err != nil {
		fail("begin tx", err)
	}
	rolledBack := false
	defer func() {
		if !rolledBack {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `SET LOCAL session_replication_role = replica`); err != nil {
		fail("disable triggers", err)
	}

	if err := m.truncateAll(ctx, tx); err != nil {
		fail("truncate", err)
	}

	for i := 0; i < len(m.tables); i++ {
		t := m.tables[i]
		n, err := m.migrateTable(ctx, tx, t)
		if err != nil {
			fail("migrate "+t.Name, err)
		}
		if *verbose || n > 0 {
			fmt.Printf("  %-40s %d rows\n", t.Name, n)
		}
	}

	if err := m.resetSequences(ctx, tx); err != nil {
		fail("reset sequences", err)
	}

	if err := m.verifyCounts(ctx, tx); err != nil {
		fail("verify", err)
	}

	if err := tx.Commit(); err != nil {
		fail("commit", err)
	}
	rolledBack = true

	fmt.Println("done.")
}

func fail(stage string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", stage, err)
	os.Exit(1)
}

func (m *migrator) loadTargetSchema() error {
	rows, err := m.dst.Query(`
		SELECT c.table_name, c.column_name, c.data_type, c.is_nullable, c.is_identity
		FROM information_schema.columns c
		JOIN information_schema.tables t
		  ON t.table_schema = c.table_schema AND t.table_name = c.table_name
		WHERE c.table_schema = 'public'
		  AND t.table_type = 'BASE TABLE'
		  AND t.table_name <> 'goose_db_version'
		ORDER BY c.table_name, c.ordinal_position
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	byTable := map[string]*tableInfo{}
	order := []string{}
	for rows.Next() {
		var (
			tbl, col, dataType, isNullable, isIdentity string
		)
		if err := rows.Scan(&tbl, &col, &dataType, &isNullable, &isIdentity); err != nil {
			return err
		}
		t, ok := byTable[tbl]
		if !ok {
			t = &tableInfo{Name: tbl}
			byTable[tbl] = t
			order = append(order, tbl)
		}
		t.Columns = append(t.Columns, columnInfo{
			Name:     col,
			DataType: dataType,
			Nullable: isNullable == "YES",
		})
		if isIdentity == "YES" {
			t.HasIdentityCol = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := 0; i < len(order); i++ {
		m.tables = append(m.tables, *byTable[order[i]])
	}
	return nil
}

func (m *migrator) countSourceRows() error {
	for i := 0; i < len(m.tables); i++ {
		t := m.tables[i]
		exists, err := m.sqliteTableExists(t.Name)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		var n int
		if err := m.src.QueryRow("SELECT COUNT(*) FROM " + quoteIdent(t.Name)).Scan(&n); err != nil {
			return fmt.Errorf("count source %s: %w", t.Name, err)
		}
		m.srcCounts[t.Name] = n
	}
	return nil
}

func (m *migrator) truncateAll(ctx context.Context, tx *sql.Tx) error {
	if len(m.tables) == 0 {
		return nil
	}
	names := make([]string, len(m.tables))
	for i := 0; i < len(m.tables); i++ {
		names[i] = quoteIdent(m.tables[i].Name)
	}
	stmt := "TRUNCATE TABLE " + strings.Join(names, ", ") + " RESTART IDENTITY CASCADE"
	_, err := tx.ExecContext(ctx, stmt)
	return err
}

func (m *migrator) migrateTable(ctx context.Context, tx *sql.Tx, t tableInfo) (int, error) {
	exists, err := m.sqliteTableExists(t.Name)
	if err != nil {
		return 0, fmt.Errorf("check source table: %w", err)
	}
	if !exists {
		return 0, nil
	}

	srcCols, err := m.sqliteColumnSet(t.Name)
	if err != nil {
		return 0, fmt.Errorf("read source columns: %w", err)
	}

	cols := make([]columnInfo, 0, len(t.Columns))
	for i := 0; i < len(t.Columns); i++ {
		c := t.Columns[i]
		if _, ok := srcCols[c.Name]; ok {
			cols = append(cols, c)
		}
	}
	if len(cols) == 0 {
		return 0, nil
	}

	colNames := make([]string, len(cols))
	for i := 0; i < len(cols); i++ {
		colNames[i] = quoteIdent(cols[i].Name)
	}
	selectSQL := "SELECT " + strings.Join(colNames, ", ") + " FROM " + quoteIdent(t.Name)

	const (
		targetPlaceholders = 10000
		maxBatchRows       = 500
	)
	batchSize := targetPlaceholders / len(cols)
	if batchSize < 1 {
		batchSize = 1
	}
	if batchSize > maxBatchRows {
		batchSize = maxBatchRows
	}

	fullStmtSQL := buildBatchInsert(t.Name, colNames, batchSize)
	fullStmt, err := tx.PrepareContext(ctx, fullStmtSQL)
	if err != nil {
		return 0, fmt.Errorf("prepare batch insert: %w", err)
	}
	defer fullStmt.Close()

	rows, err := m.src.Query(selectSQL)
	if err != nil {
		return 0, fmt.Errorf("select source: %w", err)
	}
	defer rows.Close()

	scans := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := 0; i < len(cols); i++ {
		ptrs[i] = &scans[i]
	}

	batch := make([]any, 0, batchSize*len(cols))
	count := 0

	flushFull := func() error {
		if _, err := fullStmt.ExecContext(ctx, batch...); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return count, fmt.Errorf("scan row %d: %w", count, err)
		}
		for i := 0; i < len(cols); i++ {
			converted, err := convert(scans[i], cols[i])
			if err != nil {
				return count, fmt.Errorf("row %d col %s: %w", count, cols[i].Name, err)
			}
			batch = append(batch, converted)
		}
		count++
		if len(batch)/len(cols) == batchSize {
			if err := flushFull(); err != nil {
				return count, fmt.Errorf("flush batch ending at row %d: %w", count, err)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return count, fmt.Errorf("iterate: %w", err)
	}

	if remaining := len(batch) / len(cols); remaining > 0 {
		tailSQL := buildBatchInsert(t.Name, colNames, remaining)
		if _, err := tx.ExecContext(ctx, tailSQL, batch...); err != nil {
			return count, fmt.Errorf("flush tail batch (%d rows): %w", remaining, err)
		}
	}
	return count, nil
}

func buildBatchInsert(table string, quotedCols []string, rows int) string {
	groups := make([]string, rows)
	arg := 1
	for r := 0; r < rows; r++ {
		ph := make([]string, len(quotedCols))
		for c := 0; c < len(quotedCols); c++ {
			ph[c] = fmt.Sprintf("$%d", arg)
			arg++
		}
		groups[r] = "(" + strings.Join(ph, ", ") + ")"
	}
	return "INSERT INTO " + quoteIdent(table) +
		" (" + strings.Join(quotedCols, ", ") + ") VALUES " +
		strings.Join(groups, ", ")
}

func (m *migrator) sqliteTableExists(name string) (bool, error) {
	var got string
	err := m.src.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`,
		name,
	).Scan(&got)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *migrator) sqliteColumnSet(table string) (map[string]struct{}, error) {
	rows, err := m.src.Query("PRAGMA table_info(" + quoteIdent(table) + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]struct{}{}
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		out[name] = struct{}{}
	}
	return out, rows.Err()
}

func (m *migrator) resetSequences(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT c.table_name, c.column_name
		FROM information_schema.columns c
		JOIN information_schema.tables t
		  ON t.table_schema = c.table_schema AND t.table_name = c.table_name
		WHERE c.table_schema = 'public'
		  AND t.table_type = 'BASE TABLE'
		  AND c.is_identity = 'YES'
	`)
	if err != nil {
		return err
	}

	type idCol struct {
		Table  string
		Column string
	}
	var idCols []idCol
	for rows.Next() {
		var c idCol
		if err := rows.Scan(&c.Table, &c.Column); err != nil {
			rows.Close()
			return err
		}
		idCols = append(idCols, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for i := 0; i < len(idCols); i++ {
		c := idCols[i]
		_, err := tx.ExecContext(ctx, fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s', '%s'), COALESCE((SELECT MAX(%s)+1 FROM %s), 1), false)`,
			c.Table, c.Column, quoteIdent(c.Column), quoteIdent(c.Table),
		))
		if err != nil {
			return fmt.Errorf("reset sequence for %s.%s: %w", c.Table, c.Column, err)
		}
	}
	return nil
}

func (m *migrator) verifyCounts(ctx context.Context, tx *sql.Tx) error {
	mismatches := []string{}
	for i := 0; i < len(m.tables); i++ {
		t := m.tables[i]
		srcCount, ok := m.srcCounts[t.Name]
		if !ok {
			continue
		}
		var dstCount int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+quoteIdent(t.Name)).Scan(&dstCount); err != nil {
			return fmt.Errorf("count dest %s: %w", t.Name, err)
		}
		if srcCount != dstCount {
			mismatches = append(mismatches, fmt.Sprintf("%s: src=%d dst=%d", t.Name, srcCount, dstCount))
		}
	}
	if len(mismatches) == 0 {
		fmt.Println("  all row counts match")
		return nil
	}
	sort.Strings(mismatches)
	for i := 0; i < len(mismatches); i++ {
		fmt.Println("  MISMATCH:", mismatches[i])
	}
	return fmt.Errorf("%d table(s) have row-count mismatches", len(mismatches))
}

func convert(v any, col columnInfo) (any, error) {
	if v == nil {
		return nil, nil
	}

	if s, ok := v.(string); ok && !utf8.ValidString(s) {
		v = strings.ToValidUTF8(s, "�")
	} else if b, ok := v.([]byte); ok && !utf8.Valid(b) {
		v = []byte(strings.ToValidUTF8(string(b), "�"))
	}

	switch col.DataType {
	case "boolean":
		switch x := v.(type) {
		case int64:
			return x != 0, nil
		case bool:
			return x, nil
		case string:
			if x == "1" || strings.EqualFold(x, "true") || strings.EqualFold(x, "t") {
				return true, nil
			}
			return false, nil
		default:
			return v, nil
		}

	case "timestamp with time zone", "timestamp without time zone":
		switch x := v.(type) {
		case time.Time:
			return x, nil
		case string:
			t, err := parseTimestamp(x)
			if err != nil {
				return nil, fmt.Errorf("parse timestamp %q: %w", x, err)
			}
			return t, nil
		default:
			return v, nil
		}

	case "uuid", "jsonb", "json", "USER-DEFINED":
		return v, nil

	default:
		return v, nil
	}
}

func parseTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("empty")
	}
	for i := 0; i < len(timestampFormats); i++ {
		if t, err := time.Parse(timestampFormats[i], s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("no matching format for %q", s)
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
