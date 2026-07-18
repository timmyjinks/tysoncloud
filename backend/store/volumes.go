package store

import (
	"encoding/json"
	"time"
)

type VolumesTable struct {
	ID        string    `json:"id,omitempty"`
	ServiceId string    `json:"service_id,omitempty"`
	Name      string    `json:"name,omitempty"`
	StorageGB int32     `json:"storage_gb,omitempty"`
	MountPath string    `json:"mount_path,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *SupabaseStore) GetVolumes() ([]VolumesTable, error) {
	res, _, err := s.cli.From("volumes").Select("*", "exact", false).Execute()
	if err != nil {
		return nil, err
	}

	var table []VolumesTable
	if err := json.Unmarshal([]byte(res), &table); err != nil {
		return nil, err
	}

	return table, nil
}
