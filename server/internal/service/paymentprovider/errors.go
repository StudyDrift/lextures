package paymentprovider

import "fmt"

// ErrUnknownProvider reports an unsupported webhook provider name.
func ErrUnknownProvider(name string) error {
	return fmt.Errorf("paymentprovider: unknown provider %q", name)
}
