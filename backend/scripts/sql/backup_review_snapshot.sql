SELECT concat_ws(
    chr(124),
    (SELECT COUNT(*)::text FROM mastery_objectives),
    (SELECT COUNT(*)::text FROM review_sessions),
    (SELECT COUNT(*)::text FROM review_turns),
    (SELECT COALESCE(EXTRACT(EPOCH FROM MAX(updated_at))::text, chr(45)) FROM mastery_objectives),
    (SELECT COALESCE(EXTRACT(EPOCH FROM MAX(updated_at))::text, chr(45)) FROM review_sessions),
    (SELECT COALESCE(EXTRACT(EPOCH FROM MAX(updated_at))::text, chr(45)) FROM review_turns),
    (SELECT MAX(version_id)::text FROM inkwords_review_schema_migrations WHERE is_applied)
);
