# nasty-go

Go client library for the [NASty](https://github.com/nasty-project/nasty) storage API. Communicates over WebSocket using JSON-RPC 2.0.

Used by the [NASty CSI driver](https://github.com/nasty-project/nasty-csi) to manage filesystems, subvolumes, snapshots, and sharing protocols (NFS, SMB, iSCSI, NVMe-oF) on a NASty appliance.

## Usage

```go
import nastygo "github.com/nasty-project/nasty-go"

client, err := nastygo.NewClient("wss://nasty.local/ws", "your-api-key", true)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Create a subvolume
subvol, err := client.CreateSubvolume(ctx, nastygo.SubvolumeCreateParams{
    Filesystem:    "storage",
    Name:          "my-volume",
    SubvolumeType: "filesystem",
})
```

## Retry safety

The client reconnects automatically, but only explicitly allowlisted read-only
RPCs are replayed after a lost response. Requests not yet attempted may be retried.
Mutations and unknown methods are **not** assumed idempotent, including updates
and deletes.

If a mutation may have reached the server but its response is lost, times out,
or cannot be decoded, its error wraps `nastygo.ErrUncertainOutcome`. Check with
`errors.Is(err, nastygo.ErrUncertainOutcome)` and reconcile server state before
retrying (for example, look up the requested subvolume by its stable identity).
This error does not mean the operation failed or was rolled back. It also wraps
the underlying cause, including context cancellation/deadline errors.

A definitive API error is returned unchanged. Cancellation before attempting a
request does not produce an uncertain-outcome error. Error messages do not include
request parameters, results, or authentication tokens.

## Creation-time properties (draft appliance API)

`CreateSubvolumeAtomic`, `CloneSubvolumeAtomic` and `CloneSnapshotAtomic` accept
initial user properties and call the corresponding `_atomic` RPC methods. They
require the appliance's staged-publication API; an unsupported method is an error,
not a reason to fall back to legacy creation followed by property writes.

The optional `AtomicProvisioner` interface leaves `ClientInterface` unchanged.
Atomic clones replace inherited user properties with the destination properties;
do not rely on a source volume's CSI identity/presentation IDs being inherited.
Nil properties are serialized as an empty object, not JSON null.

These mutations retain the same `ErrUncertainOutcome` behavior described above.
After a lost response, reconcile the final destination. An absent final path
does not prove no allocation occurred: interrupted private reservations require
explicit appliance-side recovery. There is no automatic adoption, repair or
deletion here. This SDK extension alone does not switch CSI provisioning or prove
bcachefs crash/power-loss durability.

## Structure

- `client.go` — WebSocket client with reconnection and JSON-RPC transport
- `interface.go` — API types and client interface definition
- `properties.go` — CSI xattr property keys and helpers
- `provisioning.go` — optional staged-publication RPC extension
- `dashboard/` — In-cluster CSI dashboard data types
