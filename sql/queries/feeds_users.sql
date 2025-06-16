-- name: ListFeedsAndUsers :many
SELECT 
    *,
    (
        SELECT name
        FROM users
        WHERE id = user_id
    ) AS created_by_user 
FROM feeds;

-- name: CreateFeedFollow :one
INSERT INTO feed_follows (
    id,
    created_at,
    updated_at,
    user_id,
    feed_id
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
) 
RETURNING 
    *,
    (
        SELECT name
        FROM users
        WHERE id = $4
    ) AS user_name,
    (
        SELECT name
        FROM feeds
        WHERE id = $5
    ) AS feed_name;

-- name: GetFeedFollowsForUser :many
SELECT 
    feeds.name AS feed_name,
    users.name AS user_name
FROM feed_follows
INNER JOIN users ON users.id = feed_follows.user_id 
INNER JOIN feeds ON feeds.id = feed_follows.feed_id
WHERE feed_follows.user_id = $1;


-- name: UnfollowFeedByURL :exec
DELETE FROM feed_follows
USING feeds
WHERE feed_follows.feed_id = feeds.id
AND feed_follows.user_id = $1
AND feeds.url = $2;