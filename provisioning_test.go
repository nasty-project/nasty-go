package nastygo

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestAtomicParamsPreserveInitialIdentity(t *testing.T) {
	properties := map[string]string{"nasty-csi:managed_by": "nasty-csi", "nasty-csi:delete_strategy": "retain"}
	for _, request := range []any{
		atomicCreateParams{SubvolumeCreateParams: SubvolumeCreateParams{Filesystem: "tank", Name: "pvc"}, InitialProperties: properties},
		atomicSnapshotCloneParams{SnapshotCloneParams: SnapshotCloneParams{Filesystem: "tank", NewName: "pvc"}, InitialProperties: properties},
	} {
		data, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			Filesystem        string            `json:"filesystem"`
			InitialProperties map[string]string `json:"initial_properties"`
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Filesystem != "tank" || !reflect.DeepEqual(decoded.InitialProperties, properties) {
			t.Fatalf("initial properties lost: %s", data)
		}
	}
}

func TestAtomicMutationsDoNotReplayOrDowngrade(t *testing.T) {
	operations := map[string]func(context.Context, *Client) error{
		"create": func(ctx context.Context, client *Client) error {
			_, err := client.CreateSubvolumeAtomic(ctx, SubvolumeCreateParams{Filesystem: "tank", Name: "pvc"}, nil)
			return err
		},
		"clone": func(ctx context.Context, client *Client) error {
			_, err := client.CloneSubvolumeAtomic(ctx, "tank", "source", "pvc", nil)
			return err
		},
		"snapshot": func(ctx context.Context, client *Client) error {
			_, err := client.CloneSnapshotAtomic(ctx, SnapshotCloneParams{Filesystem: "tank", NewName: "pvc"}, nil)
			return err
		},
	}
	for name, operation := range operations {
		for _, mode := range []string{"drop", "method-not-found"} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				client, calls := droppedResponseClient(t, "atomic-"+mode)
				err := operation(t.Context(), client)
				if err == nil || calls.Load() != 1 {
					t.Fatalf("mutation replayed/downgraded: calls=%d, error=%v", calls.Load(), err)
				}
				if mode == "drop" && !errors.Is(err, ErrUncertainOutcome) {
					t.Fatalf("uncertain cause lost: %v", err)
				}
				if mode == "method-not-found" && errors.Is(err, ErrUncertainOutcome) {
					t.Fatalf("known rejection treated as uncertain: %v", err)
				}
			})
		}
	}
}
