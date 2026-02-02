// Package notifier provides an interface for sending notifications.
package notifier

// Notifier is an interface for sending notifications.
type Notifier interface {
	Send(message string) error
}
