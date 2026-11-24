package suggestions

import "sakukota-go/internal/domain"

// Filter narrows the suggestions returned by List. The zero value applies no filter.
type Filter struct {
	// Status keeps only the suggestions in this status. Empty keeps all of them.
	Status domain.Status
}
