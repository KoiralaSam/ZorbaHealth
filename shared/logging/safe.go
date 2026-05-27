package logging

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
)

func HashIdentifier(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func SafeKV(parts ...any) string {
	if len(parts) == 0 {
		return ""
	}
	out := ""
	for i := 0; i < len(parts); i += 2 {
		if i > 0 {
			out += " "
		}
		key := fmt.Sprint(parts[i])
		value := ""
		if i+1 < len(parts) {
			value = fmt.Sprint(parts[i+1])
		}
		out += key + "=" + value
	}
	return out
}

func Info(msg string, parts ...any) {
	if len(parts) == 0 {
		log.Printf("%s", msg)
		return
	}
	log.Printf("%s %s", msg, SafeKV(parts...))
}

func Error(msg string, err error, parts ...any) {
	fields := append([]any{"error", err}, parts...)
	log.Printf("%s %s", msg, SafeKV(fields...))
}
