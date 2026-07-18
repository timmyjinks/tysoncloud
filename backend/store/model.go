package store

import (
	"github.com/supabase-community/supabase-go"
)

type SupabaseStore struct {
	cli *supabase.Client
}

func NewSupabaseStore(cli *supabase.Client) *SupabaseStore {
	return &SupabaseStore{
		cli: cli,
	}
}
