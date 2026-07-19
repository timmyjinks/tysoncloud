package store

import (
	"encoding/json"
	"time"
)

type VolumesTable struct {
	Id        string    `json:"id"`
	ServiceId string    `json:"service_id"`
	StorageGB int32     `json:"storage_gb"`
	MountPath string    `json:"mount_path"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *SupabaseStore) GetVolume(serviceId string) (VolumesTable, error) {
	res, _, err := s.cli.From("volumes").Select("*", "exact", false).Limit(1, "").Eq("service_id", serviceId).Single().Execute()
	if err != nil {
		return VolumesTable{}, err
	}

	var table VolumesTable
	if err := json.Unmarshal([]byte(res), &table); err != nil {
		return VolumesTable{}, err
	}

	return table, nil
}

func (s *SupabaseStore) CreateVolume(serviceId, mountPath string, storageGB int32) (VolumesTable, error) {
	bytes, _, err := s.cli.From("volumes").Insert(struct {
		ServiceId string `json:"service_id"`
		MountPath string `json:"mount_path"`
		StorageGB int32  `json:"storage_gb"`
	}{
		ServiceId: serviceId,
		MountPath: mountPath,
		StorageGB: storageGB,
	}, false, "", "", "").Execute()
	if err != nil {
		return VolumesTable{}, err
	}

	var res []VolumesTable = []VolumesTable{}
	if err := json.Unmarshal(bytes, &res); err != nil {
		return VolumesTable{}, err
	}

	return res[0], nil
}

func (s *SupabaseStore) DeleteVolume(serviceId string) error {
	_, _, err := s.cli.From("volumes").Delete("", "").Eq("service_id", serviceId).Execute()
	if err != nil {
		return err
	}

	return nil
}
