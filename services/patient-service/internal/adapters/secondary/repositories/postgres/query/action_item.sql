-- name: CreateActionItem :one
INSERT INTO action_items (
  call_id,
  task_description,
  is_completed,
  due_date,
  created_at
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetActionItemByID :one
SELECT * FROM action_items
WHERE id = $1 LIMIT 1;

-- name: ListActionItemsByCallID :many
SELECT * FROM action_items
WHERE call_id = $1
ORDER BY created_at DESC;

-- name: GetPendingActionItems :many
SELECT * FROM action_items
WHERE is_completed = false
ORDER BY due_date ASC NULLS LAST
LIMIT $1 OFFSET $2;

-- name: GetOverdueActionItems :many
SELECT * FROM action_items
WHERE is_completed = false
  AND due_date < NOW()
ORDER BY due_date ASC;

-- name: GetActionItemsDueByDate :many
SELECT * FROM action_items
WHERE is_completed = false
  AND due_date <= $1
ORDER BY due_date ASC;

-- name: GetActionItemsByPatient :many
SELECT ai.*
FROM action_items ai
INNER JOIN calls c ON ai.call_id = c.id
WHERE c.patient_id = $1
ORDER BY ai.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPendingActionItemsByPatient :many
SELECT ai.*
FROM action_items ai
INNER JOIN calls c ON ai.call_id = c.id
WHERE c.patient_id = $1
  AND ai.is_completed = false
ORDER BY ai.due_date ASC NULLS LAST;

-- name: UpdateActionItem :one
UPDATE action_items
SET
  task_description = COALESCE(sqlc.narg('task_description'), task_description),
  due_date = COALESCE(sqlc.narg('due_date'), due_date),
  is_completed = COALESCE(sqlc.narg('is_completed'), is_completed)
WHERE id = $1
RETURNING *;

-- name: MarkActionItemComplete :one
UPDATE action_items
SET is_completed = true
WHERE id = $1
RETURNING *;

-- name: MarkActionItemIncomplete :one
UPDATE action_items
SET is_completed = false
WHERE id = $1
RETURNING *;

-- name: DeleteActionItem :exec
DELETE FROM action_items
WHERE id = $1;

-- name: CountActionItemsByCallID :one
SELECT COUNT(*) FROM action_items
WHERE call_id = $1;

-- name: CountPendingActionItems :one
SELECT COUNT(*) FROM action_items
WHERE is_completed = false;

-- name: GetActionItemWithCallInfo :one
SELECT 
  ai.*,
  c.patient_id,
  c.livekit_room_id,
  c.started_at as call_started_at,
  p.full_name as patient_name,
  p.phone_number as patient_phone
FROM action_items ai
INNER JOIN calls c ON ai.call_id = c.id
INNER JOIN patients p ON c.patient_id = p.id
WHERE ai.id = $1
LIMIT 1;

-- name: BulkCreateActionItems :copyfrom
INSERT INTO action_items (
  call_id,
  task_description,
  is_completed,
  due_date,
  created_at
) VALUES (
  $1, $2, $3, $4, $5
);
