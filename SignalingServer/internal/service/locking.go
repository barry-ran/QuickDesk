package service

import "gorm.io/gorm/clause"

// lockForUpdate returns a SELECT ... FOR UPDATE clause for row-level
// pessimistic locking in PostgreSQL. Used in device binding transactions
// to guarantee two concurrent `POST /v1/me/devices` calls never both
// succeed on the same device row (§2.14).
func lockForUpdate() clause.Expression {
	return clause.Locking{Strength: "UPDATE"}
}
