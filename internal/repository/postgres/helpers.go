package postgres

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// UUID conversion helpers

// uuidToPgtype converts uuid.UUID to pgtype.UUID.
// If id is uuid.Nil, returns an invalid pgtype.UUID (SQL NULL).
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	var pgID pgtype.UUID
	if id == uuid.Nil {
		// For uuid.Nil, create invalid pgtype.UUID which maps to SQL NULL
		pgID.Valid = false
	} else {
		// For valid UUIDs, set the bytes and mark as valid
		pgID.Bytes = id
		pgID.Valid = true
	}
	return pgID
}

// pgtypeToUUID converts pgtype.UUID to uuid.UUID.
func pgtypeToUUID(pgID pgtype.UUID) (uuid.UUID, error) {
	if !pgID.Valid {
		return uuid.Nil, fmt.Errorf("invalid UUID")
	}
	return pgID.Bytes, nil
}

// Timestamp conversion helpers

// pgtypeToTime converts pgtype.Timestamp to time.Time.
func pgtypeToTime(ts pgtype.Timestamp) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}

// pgtypeToTimePtr converts pgtype.Timestamp to *time.Time.
func pgtypeToTimePtr(ts pgtype.Timestamp) *time.Time {
	if !ts.Valid {
		return nil
	}
	return &ts.Time
}

// timeToPgtype converts time.Time to pgtype.Timestamp.
func timeToPgtype(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{
		Time:  t,
		Valid: !t.IsZero(),
	}
}

// timePtrToPgtype converts *time.Time to pgtype.Timestamp.
func timePtrToPgtype(t *time.Time) pgtype.Timestamp {
	if t == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{
		Time:  *t,
		Valid: true,
	}
}

// JSON helpers

// emptyJSONObject returns a byte slice representing an empty JSON object.
// Use this for consistent empty JSON handling across repositories.
func emptyJSONObject() []byte {
	return []byte("{}")
}
