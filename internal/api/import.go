package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/prenansantana/waid/internal/model"
	"github.com/prenansantana/waid/pkg/phone"
)

// parseContacts detects CSV vs JSON by filename extension (falling back to
// content-type) and returns a slice of model.Contact ready for BulkUpsert.
func parseContacts(r io.Reader, filename, contentType string) ([]model.Contact, error) {
	isCSV := strings.HasSuffix(strings.ToLower(filename), ".csv") ||
		strings.Contains(contentType, "text/csv") ||
		strings.Contains(contentType, "text/plain")

	if isCSV {
		return parseCSV(r)
	}
	return parseJSON(r)
}

// parseCSV reads a CSV stream with at least a "phone" column.
// Optional columns: name, external_id.
func parseCSV(r io.Reader) ([]model.Contact, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true

	headers, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("import: reading csv header: %w", err)
	}

	idx := make(map[string]int)
	for i, h := range headers {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	phoneCol, ok := idx["phone"]
	if !ok {
		return nil, fmt.Errorf("import: csv missing required 'phone' column")
	}

	var contacts []model.Contact
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("import: reading csv row: %w", err)
		}

		raw := row[phoneCol]
		normalized, err := phone.Normalize(raw)
		if err != nil {
			continue // skip rows with invalid phone
		}

		name := ""
		if col, ok := idx["name"]; ok && col < len(row) {
			name = strings.TrimSpace(row[col])
		}

		c := model.NewContact(normalized, name)

		if col, ok := idx["external_id"]; ok && col < len(row) {
			if v := strings.TrimSpace(row[col]); v != "" {
				c.ExternalID = &v
			}
		}

		contacts = append(contacts, *c)
	}
	return contacts, nil
}

// jsonRow is the expected shape of each element in a JSON import array.
type jsonRow struct {
	Phone      string          `json:"phone"`
	Name       string          `json:"name"`
	ExternalID string          `json:"external_id"`
	Metadata   json.RawMessage `json:"metadata"`
}

// parseJSON reads a JSON array of contact objects.
func parseJSON(r io.Reader) ([]model.Contact, error) {
	var rows []jsonRow
	if err := json.NewDecoder(r).Decode(&rows); err != nil {
		return nil, fmt.Errorf("import: decoding json: %w", err)
	}

	var contacts []model.Contact
	for _, row := range rows {
		normalized, err := phone.Normalize(row.Phone)
		if err != nil {
			continue // skip rows with invalid phone
		}

		c := model.NewContact(normalized, row.Name)
		if row.ExternalID != "" {
			c.ExternalID = &row.ExternalID
		}
		if len(row.Metadata) > 0 {
			c.Metadata = row.Metadata
		}
		contacts = append(contacts, *c)
	}
	return contacts, nil
}
