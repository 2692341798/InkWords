-- +goose Up
-- InkWords migration role: core
-- Why: a BookBuild must bind the rendering toolchain as well as manuscript
-- inputs, otherwise a changed renderer could masquerade as the same build.
ALTER TABLE textbook_book_builds
    ADD COLUMN tool_versions_json JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
-- Tool versions are part of an immutable publication manifest. Refuse to
-- discard them once any book build has been recorded; restore a backup instead.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM textbook_book_builds) THEN
        RAISE EXCEPTION 'refusing to roll back book build tool versions after data exists; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd
ALTER TABLE textbook_book_builds DROP COLUMN tool_versions_json;
