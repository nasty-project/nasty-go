// Package nastygo provides a WebSocket client for the NASty storage API.
package nastygo

import "strconv"

// Schema versioning for future migrations.
const (
	// PropertySchemaVersion stores the metadata schema version.
	// Value: "1" for Schema v1.
	PropertySchemaVersion = "nasty-csi:schema_version"

	// SchemaVersionV1 is the current schema version.
	SchemaVersionV1 = "1"
)

// Xattr Property Constants - Schema v1
//
// These properties are stored as POSIX xattrs (user.* namespace) on bcachefs subvolumes
// to track CSI metadata. This approach provides:
// - Reliable metadata storage native to the filesystem (no sidecar files)
// - Ownership verification before deletion (prevents accidental deletion when IDs are reused)
// - Easy debugging via `getfattr -d <subvolume_path>` on NASty
// - Cross-cluster volume adoption support
//
// All properties use the "nasty-csi:" prefix to avoid conflicts with other tools.
const (
	// PropertyPrefix is the prefix for all nasty-csi xattr properties.
	PropertyPrefix = "nasty-csi:"

	// PropertyManagedBy indicates this resource is managed by nasty-csi.
	// Value: "nasty-csi".
	PropertyManagedBy = "nasty-csi:managed_by"

	// PropertyCSIVolumeName stores the CSI volume name (PVC name).
	// Value: e.g., "pvc-12345678-1234-1234-1234-123456789012".
	PropertyCSIVolumeName = "nasty-csi:csi_volume_name"

	// PropertyCapacityBytes stores the volume capacity in bytes.
	// Value: e.g., "10737418240" for 10GiB.
	PropertyCapacityBytes = "nasty-csi:capacity_bytes"

	// PropertyProtocol stores the storage protocol used.
	// Value: "nfs", "nvmeof", "iscsi", or "smb".
	PropertyProtocol = "nasty-csi:protocol"

	// PropertyDeleteStrategy stores the deletion strategy for the volume.
	// Value: "delete" (default) or "retain".
	// When "retain", the volume will not be deleted when the PVC is deleted.
	PropertyDeleteStrategy = "nasty-csi:delete_strategy"

	// PropertyCreatedAt stores the timestamp when the volume was created.
	// Value: RFC3339 timestamp, e.g., "2024-01-15T10:30:00Z".
	PropertyCreatedAt = "nasty-csi:created_at"
)

// Adoption metadata properties - for cross-cluster volume adoption.
const (
	// PropertyAdoptable marks a volume as adoptable by a new cluster.
	// When set to "true", CreateVolume will automatically adopt this volume
	// if found by CSI volume name, re-creating any missing NASty resources.
	// Value: "true" or "false".
	PropertyAdoptable = "nasty-csi:adoptable"

	// PropertyPVCName stores the original PVC name for adoption.
	// Value: e.g., "my-data".
	PropertyPVCName = "nasty-csi:pvc_name"

	// PropertyPVCNamespace stores the original PVC namespace for adoption.
	// Value: e.g., "default".
	PropertyPVCNamespace = "nasty-csi:pvc_namespace"

	// PropertyStorageClass stores the original StorageClass name for adoption.
	// Value: e.g., "nasty-nfs".
	PropertyStorageClass = "nasty-csi:storage_class"
)

// Multi-cluster isolation properties.
const (
	// PropertyClusterID stores the cluster identifier for multi-cluster NASty sharing.
	// When multiple K8s clusters share a NASty box, this property distinguishes
	// which cluster owns each volume/snapshot.
	// Value: user-defined cluster identifier, e.g., "prod-east", "staging".
	PropertyClusterID = "nasty-csi:cluster_id"
)

// Snapshot-specific properties.
const (
	// PropertySourceVolumeID stores the source volume ID for snapshots.
	// Value: e.g., "pvc-12345678-1234-1234-1234-123456789012".
	PropertySourceVolumeID = "nasty-csi:source_volume_id"
)

// Property values.
const (
	// ManagedByValue is the value stored in PropertyManagedBy.
	ManagedByValue = "nasty-csi"

	// ProtocolNFS indicates NFS protocol.
	ProtocolNFS = "nfs"

	// ProtocolNVMeOF indicates NVMe-oF protocol.
	ProtocolNVMeOF = "nvmeof"

	// ProtocolISCSI indicates iSCSI protocol.
	ProtocolISCSI = "iscsi"

	// ProtocolSMB indicates SMB/CIFS protocol.
	ProtocolSMB = "smb"

	// DeleteStrategyDelete is the default strategy - volume is deleted when PVC is deleted.
	DeleteStrategyDelete = "delete"

	// DeleteStrategyRetain means the volume is retained when PVC is deleted.
	DeleteStrategyRetain = "retain"
)

// PropertyNames returns all nasty-csi property names for querying.
func PropertyNames() []string {
	return []string{
		// Schema v1 core properties
		PropertySchemaVersion,
		PropertyManagedBy,
		PropertyCSIVolumeName,
		PropertyCapacityBytes,
		PropertyProtocol,
		PropertyDeleteStrategy,
		PropertyCreatedAt,
		// Adoption properties
		PropertyAdoptable,
		PropertyPVCName,
		PropertyPVCNamespace,
		PropertyStorageClass,
		// Snapshot properties
		PropertySourceVolumeID,
		// Multi-cluster
		PropertyClusterID,
	}
}

// NFSVolumeParams contains parameters for creating NFS volume properties.
type NFSVolumeParams struct {
	VolumeID       string
	CreatedAt      string
	DeleteStrategy string
	PVCName        string
	PVCNamespace   string
	StorageClass   string
	ClusterID      string
	CapacityBytes  int64
	Adoptable      bool // Mark volume as adoptable for cross-cluster adoption
}

// NFSVolumePropertiesV1 returns Schema v1 properties for an NFS volume.
//
//nolint:dupl // Intentionally similar structure to SMB volume properties
func NFSVolumePropertiesV1(params NFSVolumeParams) map[string]string {
	props := map[string]string{
		PropertySchemaVersion:  SchemaVersionV1,
		PropertyManagedBy:      ManagedByValue,
		PropertyCSIVolumeName:  params.VolumeID,
		PropertyCapacityBytes:  int64ToString(params.CapacityBytes),
		PropertyProtocol:       ProtocolNFS,
		PropertyCreatedAt:      params.CreatedAt,
		PropertyDeleteStrategy: params.DeleteStrategy,
	}
	// Add adoption properties if provided
	if params.PVCName != "" {
		props[PropertyPVCName] = params.PVCName
	}
	if params.PVCNamespace != "" {
		props[PropertyPVCNamespace] = params.PVCNamespace
	}
	if params.StorageClass != "" {
		props[PropertyStorageClass] = params.StorageClass
	}
	if params.Adoptable {
		props[PropertyAdoptable] = "true"
	}
	if params.ClusterID != "" {
		props[PropertyClusterID] = params.ClusterID
	}
	return props
}

// NVMeOFVolumeParams contains parameters for creating NVMe-oF volume properties.
type NVMeOFVolumeParams struct {
	VolumeID       string
	CreatedAt      string
	DeleteStrategy string
	PVCName        string
	PVCNamespace   string
	StorageClass   string
	ClusterID      string
	CapacityBytes  int64
	Adoptable      bool // Mark volume as adoptable for cross-cluster adoption
}

// NVMeOFVolumePropertiesV1 returns Schema v1 properties for an NVMe-oF volume.
func NVMeOFVolumePropertiesV1(params NVMeOFVolumeParams) map[string]string {
	props := map[string]string{
		PropertySchemaVersion:  SchemaVersionV1,
		PropertyManagedBy:      ManagedByValue,
		PropertyCSIVolumeName:  params.VolumeID,
		PropertyCapacityBytes:  int64ToString(params.CapacityBytes),
		PropertyProtocol:       ProtocolNVMeOF,
		PropertyCreatedAt:      params.CreatedAt,
		PropertyDeleteStrategy: params.DeleteStrategy,
	}
	// Add adoption properties if provided
	if params.PVCName != "" {
		props[PropertyPVCName] = params.PVCName
	}
	if params.PVCNamespace != "" {
		props[PropertyPVCNamespace] = params.PVCNamespace
	}
	if params.StorageClass != "" {
		props[PropertyStorageClass] = params.StorageClass
	}
	if params.Adoptable {
		props[PropertyAdoptable] = "true"
	}
	if params.ClusterID != "" {
		props[PropertyClusterID] = params.ClusterID
	}
	return props
}

// ISCSIVolumeParams contains parameters for creating iSCSI volume properties.
type ISCSIVolumeParams struct {
	VolumeID       string
	CreatedAt      string
	DeleteStrategy string
	PVCName        string
	PVCNamespace   string
	StorageClass   string
	ClusterID      string
	CapacityBytes  int64
	Adoptable      bool // Mark volume as adoptable for cross-cluster adoption
}

// ISCSIVolumePropertiesV1 returns Schema v1 properties for an iSCSI volume.
func ISCSIVolumePropertiesV1(params ISCSIVolumeParams) map[string]string {
	props := map[string]string{
		PropertySchemaVersion:  SchemaVersionV1,
		PropertyManagedBy:      ManagedByValue,
		PropertyCSIVolumeName:  params.VolumeID,
		PropertyCapacityBytes:  int64ToString(params.CapacityBytes),
		PropertyProtocol:       ProtocolISCSI,
		PropertyCreatedAt:      params.CreatedAt,
		PropertyDeleteStrategy: params.DeleteStrategy,
	}
	// Add adoption properties if provided
	if params.PVCName != "" {
		props[PropertyPVCName] = params.PVCName
	}
	if params.PVCNamespace != "" {
		props[PropertyPVCNamespace] = params.PVCNamespace
	}
	if params.StorageClass != "" {
		props[PropertyStorageClass] = params.StorageClass
	}
	if params.Adoptable {
		props[PropertyAdoptable] = "true"
	}
	if params.ClusterID != "" {
		props[PropertyClusterID] = params.ClusterID
	}
	return props
}

// SMBVolumeParams contains parameters for creating SMB volume properties.
type SMBVolumeParams struct {
	VolumeID       string
	CreatedAt      string
	DeleteStrategy string
	PVCName        string
	PVCNamespace   string
	StorageClass   string
	ClusterID      string
	CapacityBytes  int64
	Adoptable      bool // Mark volume as adoptable for cross-cluster adoption
}

// SMBVolumePropertiesV1 returns Schema v1 properties for an SMB volume.
//
//nolint:dupl // Intentionally similar structure to NFS volume properties
func SMBVolumePropertiesV1(params SMBVolumeParams) map[string]string {
	props := map[string]string{
		PropertySchemaVersion:  SchemaVersionV1,
		PropertyManagedBy:      ManagedByValue,
		PropertyCSIVolumeName:  params.VolumeID,
		PropertyCapacityBytes:  int64ToString(params.CapacityBytes),
		PropertyProtocol:       ProtocolSMB,
		PropertyCreatedAt:      params.CreatedAt,
		PropertyDeleteStrategy: params.DeleteStrategy,
	}
	// Add adoption properties if provided
	if params.PVCName != "" {
		props[PropertyPVCName] = params.PVCName
	}
	if params.PVCNamespace != "" {
		props[PropertyPVCNamespace] = params.PVCNamespace
	}
	if params.StorageClass != "" {
		props[PropertyStorageClass] = params.StorageClass
	}
	if params.Adoptable {
		props[PropertyAdoptable] = "true"
	}
	if params.ClusterID != "" {
		props[PropertyClusterID] = params.ClusterID
	}
	return props
}

// SnapshotParams contains parameters for creating snapshot properties.
type SnapshotParams struct {
	SourceVolumeID string
	Protocol       string
	ClusterID      string
}

// SnapshotPropertiesV1 returns Schema v1 properties for a snapshot.
func SnapshotPropertiesV1(params SnapshotParams) map[string]string {
	props := map[string]string{
		PropertySchemaVersion:  SchemaVersionV1,
		PropertyManagedBy:      ManagedByValue,
		PropertySourceVolumeID: params.SourceVolumeID,
		PropertyProtocol:       params.Protocol,
		PropertyDeleteStrategy: DeleteStrategyDelete,
	}
	if params.ClusterID != "" {
		props[PropertyClusterID] = params.ClusterID
	}
	return props
}

// int64ToString converts an int64 to string for xattr property storage.
func int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

// StringToInt64 converts a string to int64, returns 0 on error.
// Exported for use in controllers when reading properties.
func StringToInt64(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

// GetSchemaVersion extracts the schema version from properties.
// Returns "0" if not set (legacy volume without schema version).
func GetSchemaVersion(props map[string]string) string {
	if v, ok := props[PropertySchemaVersion]; ok && v != "" {
		return v
	}
	return "0"
}

// IsSchemaV1 checks if properties are Schema v1.
func IsSchemaV1(props map[string]string) bool {
	return GetSchemaVersion(props) == SchemaVersionV1
}
