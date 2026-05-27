// summarizer.go
package outbound

import "context"

type Summarizer interface {
	Summarize(ctx context.Context, chunks []string, focus string) (string, error)
	AnswerQuestion(ctx context.Context, question string, chunks []string) (string, error)
}
