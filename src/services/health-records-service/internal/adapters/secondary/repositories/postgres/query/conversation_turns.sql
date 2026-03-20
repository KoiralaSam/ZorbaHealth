-- Conversation memory for agent-worker sessions

-- name: CreateConversationTurn :one
INSERT INTO conversation_turns (
  patient_id,
  session_id,
  role,
  content,
  embedding
) VALUES (
  $1, $2, $3, $4, $5::vector
)
RETURNING *;

-- name: ListRecentConversationTurns :many
SELECT *
FROM conversation_turns
WHERE patient_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ListRecentConversationTurnsBySession :many
SELECT *
FROM conversation_turns
WHERE patient_id = $1
  AND session_id = $2
ORDER BY created_at DESC
LIMIT $3;

-- name: DeleteConversationTurnsByPatientID :exec
DELETE FROM conversation_turns
WHERE patient_id = $1;

