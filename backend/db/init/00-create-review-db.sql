DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'inkwords_review_db') THEN
    CREATE DATABASE inkwords_review_db;
  END IF;
END$$;
