-- name: InsertJob
INSERT INTO scrape_jobs (url, root_host) VALUES (@url, @root_host)

-- name: PromoteAndGetJob
UPDATE scrape_jobs
SET status = 'running', started_at = CURRENT_TIMESTAMP
WHERE id = (
    SELECT id FROM scrape_jobs
    WHERE status = 'pending'
    ORDER BY created_at
    LIMIT 1
)
AND (SELECT COUNT(*) FROM scrape_jobs WHERE status = 'running') < @max_jobs
RETURNING id, url, root_host

-- name: CompleteJob
UPDATE scrape_jobs
SET status = @status, completed_at = CURRENT_TIMESTAMP, error_msg = @error_msg
WHERE id = @id

-- name: RecoverJobs
UPDATE scrape_jobs
SET status = 'error', error_msg = 'server restarted', completed_at = CURRENT_TIMESTAMP
WHERE status = 'running'

-- name: GetJob
SELECT id, url, root_host, status, created_at, started_at, completed_at, error_msg
FROM scrape_jobs
WHERE id = @id
