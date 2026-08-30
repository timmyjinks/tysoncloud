package store

import (
	"fmt"
	"strings"
)

// rpcError wraps a PostgrestError returned by an RPC call, mapping common
// constraint violations to a message a user can act on.
func rpcError(op string, pgErr PostgrestError) error {
	msg := pgErr.Message
	if friendly, ok := friendlyPostgres(msg); ok {
		msg = friendly
	}
	return fmt.Errorf("%s failed: %s", op, msg)
}

// friendlyPostgres maps common Postgres error messages to something a user can
// act on instead of a raw constraint name.
func friendlyPostgres(msg string) (string, bool) {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "duplicate key value"):
		return "That name is already in use. Pick a different one.", true
	case strings.Contains(lower, "violates check constraint"):
		return "One of the provided values isn't allowed.", true
	case strings.Contains(lower, "violates foreign key constraint"):
		return "That resource is linked to another resource and can't be changed.", true
	}
	return "", false
}
