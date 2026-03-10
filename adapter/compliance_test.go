package adapter

// Compile-time interface compliance checks.
// These ensure the Mock adapter implements every interface in the spec.

var _ Adapter = (*Mock)(nil)
var _ SessionProvider = (*Mock)(nil)
var _ HistoryClearer = (*Mock)(nil)
var _ HistoryProvider = (*Mock)(nil)
var _ ConversationManager = (*Mock)(nil)
var _ PermissionResponder = (*Mock)(nil)
var _ StatusListener = (*Mock)(nil)
