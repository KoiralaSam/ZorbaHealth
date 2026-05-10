// summarizer.go
package outbound

import "context"

type Summarizer interface {
	Summarize(ctx context.Context, chunks []string, focus string) (string, error)
}
