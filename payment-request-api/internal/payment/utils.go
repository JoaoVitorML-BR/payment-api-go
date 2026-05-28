package payment

import "github.com/jackc/pgx/v5/pgtype"

// Convert a string UUID to pgtype.UUID
func parseStringToUUID(uuidStr string) pgtype.UUID {
	var uuid pgtype.UUID
	uuid.Scan(uuidStr)
	return uuid
}