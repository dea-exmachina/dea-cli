package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/dea-exmachina/dea-cli/internal/config"
)

// QueuedRequest represents a failed API call stored for later retry.
type QueuedRequest struct {
	ID       string          `json:"id"`
	Method   string          `json:"method"`
	Path     string          `json:"path"`
	Body     json.RawMessage `json:"body"`
	QueuedAt time.Time       `json:"queued_at"`
}

// Queue manages offline request persistence at ~/.dea/queue.json.
type Queue struct {
	mu   sync.Mutex
	path string
}

// New creates a Queue backed by ~/.dea/queue.json.
func New() *Queue {
	return &Queue{path: config.QueuePath()}
}

// Add appends a request to the queue.
func (q *Queue) Add(method, path string, body interface{}) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	items, err := q.load()
	if err != nil {
		items = []QueuedRequest{}
	}

	var rawBody json.RawMessage
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal queue body: %w", err)
		}
		rawBody = data
	}

	items = append(items, QueuedRequest{
		ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
		Method:   method,
		Path:     path,
		Body:     rawBody,
		QueuedAt: time.Now().UTC(),
	})

	return q.save(items)
}

// List returns all queued requests.
func (q *Queue) List() ([]QueuedRequest, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.load()
}

// Clear removes all items from the queue.
func (q *Queue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.save([]QueuedRequest{})
}

// Remove deletes a specific item by ID.
func (q *Queue) Remove(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	items, err := q.load()
	if err != nil {
		return err
	}

	filtered := make([]QueuedRequest, 0, len(items))
	for _, item := range items {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}
	return q.save(filtered)
}

// Len returns the number of queued items.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	items, err := q.load()
	if err != nil {
		return 0
	}
	return len(items)
}

func (q *Queue) load() ([]QueuedRequest, error) {
	data, err := os.ReadFile(q.path)
	if os.IsNotExist(err) {
		return []QueuedRequest{}, nil
	}
	if err != nil {
		return nil, err
	}

	var items []QueuedRequest
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (q *Queue) save(items []QueuedRequest) error {
	if err := os.MkdirAll(config.DeaDir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(q.path, data, 0600)
}
