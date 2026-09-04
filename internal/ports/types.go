// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package ports

import (
	"context"

	"github.com/gostafa/inputstats/internal/domain"
)

type (
	// InputPort is the OS-specific input monitoring adapter.
	InputPort interface {
		Run(ctx context.Context, events chan<- domain.Event) error
	}
)
