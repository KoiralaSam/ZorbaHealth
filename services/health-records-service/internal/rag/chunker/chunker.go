package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type Chunker interface {
	Chunk(text string) []string
}

type WordChunker struct {
	chunkSize int
	overlap   int
}

func NewWordChunker(chunkSize, overlap int) WordChunker {
	if chunkSize <= overlap {
		chunkSize = overlap + 1
	}
	return WordChunker{chunkSize: chunkSize, overlap: overlap}
}

func (c WordChunker) Chunk(text string) []string {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return nil
	}
	var chunks []string
	for i := 0; i < len(words); i += c.chunkSize - c.overlap {
		end := i + c.chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[i:end], " "))
		if end == len(words) {
			break
		}
	}
	return chunks
}

func Hash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
