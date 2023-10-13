package pool

type ParameterStatusSync int

const (
	// ParameterStatusSyncNone does not attempt to sync parameter status.
	ParameterStatusSyncNone ParameterStatusSync = iota
	// ParameterStatusSyncInitial assumes both client and server have their initial status before syncing.
	// Use in session pooling for lower latency
	ParameterStatusSyncInitial
	// ParameterStatusSyncDynamic will track parameter status and ensure they are synced correctly.
	// Use in transaction pooling
	ParameterStatusSyncDynamic
)
