package jlpt

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sourcePattern = "https://raw.githubusercontent.com/stephenmk/yomitan-jlpt-vocab/main/original_data/n%d.csv"

type Importer struct {
	DB      *sql.DB
	Client  *http.Client
	BaseURL string
}

type Report struct{ Levels, Rows, Words, Links int }

func (i *Importer) Run(ctx context.Context) (Report, error) {
	client := i.Client
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	if err := i.ensureSchema(ctx); err != nil {
		return Report{}, err
	}
	report := Report{}
	for level := 5; level >= 1; level-- {
		url := fmt.Sprintf(sourcePattern, level)
		if i.BaseURL != "" {
			url = fmt.Sprintf(strings.TrimRight(i.BaseURL, "/")+"/n%d.csv", level)
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err != nil {
			return report, fmt.Errorf("download N%d: %w", level, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return report, fmt.Errorf("download N%d: HTTP %d", level, resp.StatusCode)
		}
		r, err := i.importLevel(ctx, level, resp.Body)
		resp.Body.Close()
		if err != nil {
			return report, err
		}
		report.Levels++
		report.Rows += r.Rows
		report.Words += r.Words
		report.Links += r.Links
	}
	return report, nil
}

func (i *Importer) ensureSchema(ctx context.Context) error {
	_, err := i.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS word_decks (word_id BIGINT NOT NULL, deck_id BIGINT NOT NULL, PRIMARY KEY(word_id,deck_id), FOREIGN KEY(word_id) REFERENCES words(id) ON DELETE CASCADE, FOREIGN KEY(deck_id) REFERENCES decks(id) ON DELETE CASCADE)`)
	if err != nil {
		return fmt.Errorf("prepare schema: %w", err)
	}
	return nil
}

func (i *Importer) importLevel(ctx context.Context, level int, src io.Reader) (Report, error) {
	tx, err := i.DB.BeginTx(ctx, nil)
	if err != nil {
		return Report{}, err
	}
	defer tx.Rollback()
	name := fmt.Sprintf("JLPT N%d", level)
	desc := fmt.Sprintf("Community-curated vocabulary commonly associated with JLPT N%d", level)
	var deckID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM decks WHERE name=? ORDER BY id LIMIT 1`, name).Scan(&deckID)
	if err == sql.ErrNoRows {
		res, insertErr := tx.ExecContext(ctx, `INSERT INTO decks(name,description) VALUES(?,?)`, name, desc)
		if insertErr != nil {
			return Report{}, fmt.Errorf("create %s deck: %w", name, insertErr)
		}
		deckID, err = res.LastInsertId()
	} else if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE decks SET description=? WHERE id=?`, desc, deckID)
	}
	if err != nil {
		return Report{}, err
	}
	r := csv.NewReader(src)
	r.FieldsPerRecord = -1
	if _, err = r.Read(); err != nil {
		return Report{}, fmt.Errorf("read N%d header: %w", level, err)
	}
	report := Report{}
	for line := 2; ; line++ {
		record, readErr := r.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return report, fmt.Errorf("N%d line %d: %w", level, line, readErr)
		}
		if len(record) < 4 {
			continue
		}
		seq, parseErr := strconv.ParseInt(strings.TrimSpace(record[0]), 10, 64)
		if parseErr != nil {
			continue
		}
		reading, spelling, english := strings.TrimSpace(record[1]), strings.TrimSpace(record[2]), strings.TrimSpace(record[3])
		if spelling == "" {
			spelling = reading
		}
		if reading == "" || english == "" {
			continue
		}
		res, execErr := tx.ExecContext(ctx, `INSERT INTO words(jmdict_seq,english,japanese,reading) VALUES(?,?,?,?) ON DUPLICATE KEY UPDATE english=VALUES(english),japanese=VALUES(japanese),reading=VALUES(reading)`, seq, english, spelling, reading)
		if execErr != nil {
			return report, fmt.Errorf("N%d line %d: %w", level, line, execErr)
		}
		if n, _ := res.RowsAffected(); n == 1 {
			report.Words++
		}
		var wordID int64
		if err = tx.QueryRowContext(ctx, `SELECT id FROM words WHERE jmdict_seq=?`, seq).Scan(&wordID); err != nil {
			return report, err
		}
		link, linkErr := tx.ExecContext(ctx, `INSERT IGNORE INTO word_decks(word_id,deck_id) VALUES(?,?)`, wordID, deckID)
		if linkErr != nil {
			return report, linkErr
		}
		if n, _ := link.RowsAffected(); n > 0 {
			report.Links++
		}
		report.Rows++
	}
	if err = tx.Commit(); err != nil {
		return report, err
	}
	return report, nil
}
