package database

import "database/sql"

// RunSQLMigrations creates required tables explicitly using SQL.
func RunSQLMigrations(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			first_name VARCHAR(128) NOT NULL,
			last_name VARCHAR(128) NOT NULL,
			middle_name VARCHAR(128) NOT NULL DEFAULT '',
			full_name VARCHAR(255) NOT NULL,
			username VARCHAR(64) NOT NULL UNIQUE,
			password_plain VARCHAR(255) NOT NULL,
			role VARCHAR(32) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS first_name VARCHAR(128) NOT NULL DEFAULT '';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_name VARCHAR(128) NOT NULL DEFAULT '';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS middle_name VARCHAR(128) NOT NULL DEFAULT '';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS full_name VARCHAR(255) NOT NULL DEFAULT '';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(64);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(255);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_plain VARCHAR(255) NOT NULL DEFAULT '';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(32) NOT NULL DEFAULT 'viewer';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();`,
		`ALTER TABLE users ALTER COLUMN email DROP NOT NULL;`,
		`ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS users_username_uindex ON users (username);`,
		`UPDATE users
		 SET username = CONCAT('user_', id)
		 WHERE username IS NULL OR TRIM(username) = '';`,
		`CREATE TABLE IF NOT EXISTS user_sessions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS plans (
			id BIGSERIAL PRIMARY KEY,
			year INT NOT NULL UNIQUE,
			status VARCHAR(32) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id BIGSERIAL PRIMARY KEY,
			plan_id BIGINT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
			parent_id BIGINT NULL REFERENCES tasks(id) ON DELETE SET NULL,
			title VARCHAR(255) NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			planned_value VARCHAR(64) NOT NULL DEFAULT '',
			deadline DATE NULL,
			responsible_user_id BIGINT NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'created',
			result_text TEXT NOT NULL DEFAULT '',
			completion_percent INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS reports (
			id BIGSERIAL PRIMARY KEY,
			task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			file_path VARCHAR(1024) NOT NULL,
			uploaded_by BIGINT NOT NULL,
			uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS task_logs (
			id BIGSERIAL PRIMARY KEY,
			task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			action VARCHAR(64) NOT NULL,
			old_status VARCHAR(32) NOT NULL DEFAULT '',
			new_status VARCHAR(32) NOT NULL DEFAULT '',
			user_id BIGINT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS planning_period_indicators (
			id BIGSERIAL PRIMARY KEY,
			target_indicator TEXT NOT NULL,
			unit VARCHAR(32) NOT NULL,
			year_values JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS plan_indicator_details (
			id BIGSERIAL PRIMARY KEY,
			planning_period_indicator_id BIGINT NOT NULL REFERENCES planning_period_indicators(id) ON DELETE CASCADE,
			year INT NOT NULL,
			development_indicator TEXT NOT NULL DEFAULT '',
			activities TEXT NOT NULL DEFAULT '',
			execution_deadline TEXT NOT NULL DEFAULT '',
			responsible TEXT NOT NULL DEFAULT '',
			responsible_user_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (planning_period_indicator_id, year)
		);`,
		`ALTER TABLE plan_indicator_details
		  ADD COLUMN IF NOT EXISTS responsible_user_ids JSONB NOT NULL DEFAULT '[]'::jsonb;`,
		`CREATE TABLE IF NOT EXISTS plan_indicator_reports (
			id BIGSERIAL PRIMARY KEY,
			planning_period_indicator_id BIGINT NOT NULL REFERENCES planning_period_indicators(id) ON DELETE CASCADE,
			year INT NOT NULL,
			report_text TEXT NOT NULL DEFAULT '',
			file_path VARCHAR(1024) NOT NULL DEFAULT '',
			file_name VARCHAR(255) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			review_note TEXT NOT NULL DEFAULT '',
			approval_formula TEXT NOT NULL DEFAULT '',
			reviewed_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
			reviewed_at TIMESTAMPTZ NULL,
			submitted_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (planning_period_indicator_id, year, submitted_by)
		);`,
		`ALTER TABLE plan_indicator_reports
		  ADD COLUMN IF NOT EXISTS file_name VARCHAR(255) NOT NULL DEFAULT '';`,
		`ALTER TABLE plan_indicator_reports
		  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();`,
		`ALTER TABLE plan_indicator_reports
		  ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'pending';`,
		`ALTER TABLE plan_indicator_reports
		  ADD COLUMN IF NOT EXISTS review_note TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE plan_indicator_reports
		  ADD COLUMN IF NOT EXISTS approval_formula TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE plan_indicator_reports
		  ADD COLUMN IF NOT EXISTS reviewed_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL;`,
		`ALTER TABLE plan_indicator_reports
		  ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ NULL;`,
		`UPDATE plan_indicator_reports
		 SET status = 'pending'
		 WHERE status IS NULL OR TRIM(status) = '';`,
		`CREATE TABLE IF NOT EXISTS plan_indicator_report_files (
			id BIGSERIAL PRIMARY KEY,
			report_id BIGINT NOT NULL REFERENCES plan_indicator_reports(id) ON DELETE CASCADE,
			file_name VARCHAR(255) NOT NULL,
			storage_path TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`ALTER TABLE plan_indicator_report_files
		  ADD COLUMN IF NOT EXISTS storage_path TEXT NOT NULL DEFAULT '';`,
		`INSERT INTO plan_indicator_report_files (report_id, file_name, storage_path, created_at)
		SELECT pir.id,
		       CASE
		           WHEN COALESCE(NULLIF(TRIM(pir.file_name), ''), '') <> '' THEN pir.file_name
		           ELSE CONCAT('legacy_file_', pir.id)
		       END,
		       pir.file_path,
		       NOW()
		FROM plan_indicator_reports pir
		WHERE COALESCE(TRIM(pir.file_path), '') <> ''
		  AND NOT EXISTS (
		      SELECT 1
		      FROM plan_indicator_report_files rf
		      WHERE rf.report_id = pir.id
		  );`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}
