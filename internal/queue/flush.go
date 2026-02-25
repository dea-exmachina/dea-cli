package queue

import (
	"fmt"

	"github.com/dea-exmachina/dea-cli/internal/api"
)

// Flush attempts to replay all queued requests against the API.
// Successfully replayed requests are removed from the queue.
// Returns the number of items flushed and any error encountered.
func Flush(q *Queue, client *api.Client) (int, error) {
	items, err := q.List()
	if err != nil {
		return 0, fmt.Errorf("failed to load queue: %w", err)
	}

	if len(items) == 0 {
		return 0, nil
	}

	flushed := 0
	for _, item := range items {
		var respErr error
		switch item.Method {
		case "POST":
			_, respErr = client.Post(item.Path, item.Body)
		case "GET":
			_, respErr = client.Get(item.Path)
		default:
			// Unknown method — skip and remove to avoid infinite retry.
			fmt.Printf("Skipping unsupported queued method %s %s\n", item.Method, item.Path)
			_ = q.Remove(item.ID)
			continue
		}

		if respErr != nil {
			if api.IsNetworkError(respErr) {
				// Still offline — stop flushing.
				break
			}
			// Non-network error (e.g. 4xx) — remove from queue to avoid infinite retry.
			fmt.Printf("Queued request %s failed with non-network error: %v (removing)\n", item.ID, respErr)
			_ = q.Remove(item.ID)
			continue
		}

		if err := q.Remove(item.ID); err != nil {
			fmt.Printf("Warning: failed to remove flushed item %s: %v\n", item.ID, err)
		}
		flushed++
	}

	return flushed, nil
}
