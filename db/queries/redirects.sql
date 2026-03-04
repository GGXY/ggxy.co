-- name: ListRedirects :many
SELECT *
FROM redirects
ORDER BY source_host, source_path;

-- name: GetRedirect :one
SELECT *
FROM redirects
WHERE id = ?;

-- name: CreateRedirect :one
INSERT INTO redirects (source_host, source_path, destination, redirect_type, is_regex)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateRedirect :one
UPDATE redirects
SET source_host = ?,
    source_path = ?,
    destination = ?,
    redirect_type = ?,
    is_regex = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteRedirect :exec
DELETE FROM redirects
WHERE id = ?;

-- name: IncrementHitCount :exec
UPDATE redirects
SET hit_count = hit_count + 1
WHERE id = ?;

-- name: GetRedirectsByHost :many
SELECT *
FROM redirects
WHERE source_host = ?
ORDER BY length(source_path) DESC;

-- name: GetCatchAllRedirects :many
SELECT *
FROM redirects
WHERE source_host = ''
ORDER BY length(source_path) DESC;
