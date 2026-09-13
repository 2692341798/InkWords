SELECT concat_ws(
    chr(124),
    (SELECT COUNT(*)::text FROM job_tasks),
    (SELECT COUNT(*)::text FROM textbook_projects),
    (SELECT COUNT(*)::text FROM blogs),
    (SELECT COALESCE(EXTRACT(EPOCH FROM MAX(updated_at))::text, chr(45)) FROM job_tasks),
    (SELECT COALESCE(EXTRACT(EPOCH FROM MAX(updated_at))::text, chr(45)) FROM textbook_projects),
    (SELECT COALESCE(EXTRACT(EPOCH FROM MAX(updated_at))::text, chr(45)) FROM blogs),
    (SELECT MAX(version_id)::text FROM inkwords_core_schema_migrations WHERE is_applied)
);
