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
