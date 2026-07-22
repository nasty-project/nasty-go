package nastygo

import (
	"encoding/json"
	"testing"
)

func TestSubvolumeDecodesAuthoritativeCapacities(t *testing.T) {
	const payload = `{"name":"pvc","filesystem":"tank","subvolume_type":"filesystem","path":"/fs/tank/pvc","used_bytes":1024,"quota_bytes":5368709120,"volsize_bytes":null,"block_device":null,"snapshots":[],"owner":null,"properties":{}}`

	var subvolume Subvolume
	if err := json.Unmarshal([]byte(payload), &subvolume); err != nil {
		t.Fatalf("decode subvolume: %v", err)
	}
	if subvolume.QuotaBytes == nil || *subvolume.QuotaBytes != 5*1024*1024*1024 {
		t.Fatalf("QuotaBytes = %v, want 5 GiB", subvolume.QuotaBytes)
	}
	if subvolume.VolsizeBytes != nil {
		t.Fatalf("VolsizeBytes = %v, want nil for filesystem subvolume", subvolume.VolsizeBytes)
	}
}

func TestSubvolumeCreateParamsEncodesBlockFilesystem(t *testing.T) {
	params := SubvolumeCreateParams{
		Filesystem:      "tank",
		Name:            "pvc",
		SubvolumeType:   "block",
		BlockFilesystem: "xfs",
	}

	payload, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("encode create params: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode create params: %v", err)
	}
	if decoded["block_filesystem"] != "xfs" {
		t.Fatalf("block_filesystem = %v, want xfs", decoded["block_filesystem"])
	}
}

func TestSubvolumeDecodesBlockFilesystemResult(t *testing.T) {
	const payload = `{"name":"pvc","filesystem":"tank","subvolume_type":"block","path":"/fs/tank/pvc","block_filesystem":"ext4","block_filesystem_uuid":"abc-123","created":true}`

	var subvolume Subvolume
	if err := json.Unmarshal([]byte(payload), &subvolume); err != nil {
		t.Fatalf("decode subvolume: %v", err)
	}
	if subvolume.BlockFilesystem == nil || *subvolume.BlockFilesystem != "ext4" {
		t.Fatalf("BlockFilesystem = %v, want ext4", subvolume.BlockFilesystem)
	}
	if subvolume.BlockFilesystemUUID == nil || *subvolume.BlockFilesystemUUID != "abc-123" {
		t.Fatalf("BlockFilesystemUUID = %v, want abc-123", subvolume.BlockFilesystemUUID)
	}
	if !subvolume.Created {
		t.Fatal("Created = false, want true")
	}
}

func TestSubvolumeWithoutCreatedFieldIsConservativelyExisting(t *testing.T) {
	const payload = `{"name":"pvc","filesystem":"tank","subvolume_type":"block","path":"/fs/tank/pvc"}`

	var subvolume Subvolume
	if err := json.Unmarshal([]byte(payload), &subvolume); err != nil {
		t.Fatalf("decode subvolume: %v", err)
	}
	if subvolume.Created {
		t.Fatal("Created = true for an older response without creation authority")
	}
}
