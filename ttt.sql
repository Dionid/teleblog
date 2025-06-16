-- SQLite
SELECT 
    post.*,
    COUNT(comment.id) AS comment_count
FROM 
    post
LEFT JOIN
    comment ON post.id = comment.post_id;