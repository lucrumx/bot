// Package wstopics bingx ws private channel topics
package wstopics

const (

	// PrivateExecutionEvent represents the event type for executions (order_update) ws private channel .
	PrivateExecutionEvent = "TRADE_UPDATE"
	// PrivateAccountEvent This type of event will be pushed when a new order is created, an order has a new deal, or a new status change. The event type is unified as ORDER_TRADE_UPDATE.
	PrivateAccountEvent = "ACCOUNT_UPDATE"
	// PrivateAccountConfigEvent When the account configuration changes, the event type will be pushed as ACCOUNT_CONFIG_UPDATE (leverage for example)
	PrivateAccountConfigEvent = "ACCOUNT_CONFIG_UPDATE"
)
