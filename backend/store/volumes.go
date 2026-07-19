package store

import (
	"encoding/json"
	"errors"
	"fmt"
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
	res, _, err := s.cli.From("volumes").Select("*", "exact", false).Eq("service_id", serviceId).Single().Execute()
	if err != nil {
		return VolumesTable{}, err
	}

	var table VolumesTable
	if err := json.Unmarshal([]byte(res), &table); err != nil {
		return VolumesTable{}, err
	}

	return table, nil
}

func (s *SupabaseStore) CreateVolume(serviceId, userId, mountPath string, storageGB int32) (VolumesTable, error) {
	result := s.cli.Rpc("create_volume", "", map[string]interface{}{
		"p_service_id": serviceId,
		"p_user_id":    userId,
		"p_mount_path": mountPath,
		"p_storage_gb": storageGB,
	})

	var res VolumesTable
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		return VolumesTable{}, err
	}

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return VolumesTable{}, fmt.Errorf("create_service failed: %s", pgErr.Message)
	}

	return res, nil
}

func (s *SupabaseStore) DeleteVolume(serviceId, userId string) error {
	result := s.cli.Rpc("delete_volume", "", map[string]interface{}{
		"p_service_id": serviceId,
		"p_user_id":    userId,
	})

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return fmt.Errorf("create_service failed: %s", pgErr.Message)
	}

	return nil
}
