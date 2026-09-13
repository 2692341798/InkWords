-- +goose Up
-- InkWords migration role: core
-- Why: manifest 包含构建时间，不能用 manifest_hash 判断两次冻结请求是否来自相同批准输入。
ALTER TABLE textbook_book_builds
    ADD COLUMN input_hash TEXT;

-- 旧构建没有独立输入哈希；保留其 manifest hash 作为一次性的兼容身份，
-- 不伪造无法从旧数据可靠重建的时间无关输入指纹。
UPDATE textbook_book_builds
SET input_hash = COALESCE(NULLIF(manifest_json ->> 'input_hash', ''), manifest_hash)
WHERE input_hash IS NULL;

ALTER TABLE textbook_book_builds
    ALTER COLUMN input_hash SET NOT NULL,
    ADD CONSTRAINT ck_textbook_book_builds_input_hash
        CHECK (input_hash LIKE 'sha256:%');

-- Target query: project row lock内按稳定冻结输入复用既有 Build。
-- Cost: 每个低频 BookBuild 多一次唯一 B-tree 写入和少量存储。
CREATE UNIQUE INDEX ux_textbook_book_builds_project_input
    ON textbook_book_builds (project_id, input_hash);

-- +goose Down
-- Why: 删除输入身份会恢复重复冻结缺陷；已有 Build 时只能从备份恢复。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM textbook_book_builds) THEN
        RAISE EXCEPTION 'refusing to roll back book build input identity after builds exist; restore a backup instead';
    END IF;
END
$$;
-- +goose StatementEnd

DROP INDEX ux_textbook_book_builds_project_input;
ALTER TABLE textbook_book_builds
    DROP CONSTRAINT ck_textbook_book_builds_input_hash,
    DROP COLUMN input_hash;
