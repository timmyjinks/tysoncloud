package db

import (
	"github.com/supabase-community/supabase-go"
)

func NewSupabaseStorage(url, apiKey string) (*supabase.Client, error) {
	cli, err := supabase.NewClient(url, apiKey, &supabase.ClientOptions{})
	if err != nil {
		return nil, err
	}
	return cli, nil
}
