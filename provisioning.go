package nastygo

import (
	"context"
	"fmt"
)

// AtomicProvisioner is an optional extension rather than a breaking change to
// ClientInterface. These calls require the appliance's atomic-publication API;
// unsupported methods never fall back to legacy creation or property writes.
type AtomicProvisioner interface {
	CreateSubvolumeAtomic(context.Context, SubvolumeCreateParams, map[string]string) (*Subvolume, error)
	CloneSubvolumeAtomic(context.Context, string, string, string, map[string]string) (*Subvolume, error)
	CloneSnapshotAtomic(context.Context, SnapshotCloneParams, map[string]string) (*Subvolume, error)
}

var _ AtomicProvisioner = (*Client)(nil)

func atomicProperties(properties map[string]string) map[string]string {
	if properties == nil {
		return map[string]string{}
	}
	return properties
}

type atomicCreateParams struct {
	SubvolumeCreateParams
	InitialProperties map[string]string `json:"initial_properties"`
}

type atomicSnapshotCloneParams struct {
	SnapshotCloneParams
	InitialProperties map[string]string `json:"initial_properties"`
}

// CreateSubvolumeAtomic supplies ownership/identity properties before the new
// subvolume is published. A lost response returns ErrUncertainOutcome, not replay.
func (c *Client) CreateSubvolumeAtomic(ctx context.Context, params SubvolumeCreateParams, properties map[string]string) (*Subvolume, error) {
	var result Subvolume
	request := atomicCreateParams{SubvolumeCreateParams: params, InitialProperties: atomicProperties(properties)}
	if err := c.Call(ctx, "subvolume.create_atomic", request, &result); err != nil {
		return nil, fmt.Errorf("atomic create %s/%s: %w", params.Filesystem, params.Name, err)
	}
	return &result, nil
}

// CloneSubvolumeAtomic replaces inherited user properties with the supplied
// destination identity before publication. It never relabels an existing object.
func (c *Client) CloneSubvolumeAtomic(ctx context.Context, filesystem, name, newName string, properties map[string]string) (*Subvolume, error) {
	var result Subvolume
	if err := c.Call(ctx, "subvolume.clone_atomic", map[string]interface{}{
		"filesystem": filesystem, "name": name, "new_name": newName,
		"initial_properties": atomicProperties(properties),
	}, &result); err != nil {
		return nil, fmt.Errorf("atomic clone %s/%s to %s: %w", filesystem, name, newName, err)
	}
	return &result, nil
}

// CloneSnapshotAtomic publishes a snapshot clone with destination properties,
// without inheriting the source's user identity or presentation IDs.
func (c *Client) CloneSnapshotAtomic(ctx context.Context, params SnapshotCloneParams, properties map[string]string) (*Subvolume, error) {
	var result Subvolume
	request := atomicSnapshotCloneParams{SnapshotCloneParams: params, InitialProperties: atomicProperties(properties)}
	if err := c.Call(ctx, "snapshot.clone_atomic", request, &result); err != nil {
		return nil, fmt.Errorf("atomic snapshot clone to %s/%s: %w", params.Filesystem, params.NewName, err)
	}
	return &result, nil
}
