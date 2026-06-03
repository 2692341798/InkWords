# Review 数据迁移 Runbook（core db -> inkwords_review_db）

本文档用于将 review 相关数据从 core db 迁移到 `inkwords_review_db`，并提供可回滚路径（将 `REVIEW_DATABASE_URL` 指回 core db 并重启）。

## 前置条件

- 本地已可运行 Docker / Docker Compose
- Postgres 容器已启动（默认容器名：`inkwords-db`）
- core db 已存在（默认库名：`$POSTGRES_DB`，常见为 `inkwords_db`）
- 需要迁移的表在 core db 中已存在：`review_sessions`、`review_turns`

## 当 pgdata 已存在时：手动创建 inkwords_review_db

当 Postgres 已经使用过并持久化了 `pgdata` volume 时，初始化脚本可能不会再次执行。如果 `inkwords_review_db` 尚未创建，可手动执行：

```bash
docker exec -i inkwords-db psql -U "$POSTGRES_USER" -d postgres -c "CREATE DATABASE inkwords_review_db;"
```

验证数据库已创建：

```bash
docker exec -i inkwords-db psql -U "$POSTGRES_USER" -d postgres -c "\l" | grep inkwords_review_db
```

## 数据迁移：导出 core db 两张表并导入 review db

将 core db 的 `review_sessions` 与 `review_turns` 导出为 SQL，再导入到 `inkwords_review_db`：

```bash
docker exec -i inkwords-db psql -U "$POSTGRES_USER" -d inkwords_review_db -c "TRUNCATE TABLE review_turns, review_sessions RESTART IDENTITY;"
docker exec -t inkwords-db pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --data-only -t review_sessions -t review_turns > /tmp/review_dump.sql
docker exec -i inkwords-db psql -U "$POSTGRES_USER" -d inkwords_review_db < /tmp/review_dump.sql
```

## 验证：迁移前后行数对比

分别在 core db 与 review db 查询行数，要求两边一致（两张表都要对比）：

```bash
docker exec -i inkwords-db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select count(*) from review_sessions;"
docker exec -i inkwords-db psql -U "$POSTGRES_USER" -d inkwords_review_db -c "select count(*) from review_sessions;"

docker exec -i inkwords-db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select count(*) from review_turns;"
docker exec -i inkwords-db psql -U "$POSTGRES_USER" -d inkwords_review_db -c "select count(*) from review_turns;"
```

## 回滚：将 REVIEW_DATABASE_URL 指回 core db 并重启

如果迁移后 review-service 行为异常或需要立即回退：

1. 将 `review-service` 的 `REVIEW_DATABASE_URL` 改回指向 core db（例如 `...@db:5432/${POSTGRES_DB}?sslmode=disable`）
2. 一键重启（确保配置生效）：

```bash
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
```
