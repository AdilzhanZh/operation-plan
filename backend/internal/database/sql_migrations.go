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
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ
		);`,
		`ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;`,
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
			direction TEXT NOT NULL DEFAULT 'Академическое превосходство и интернационализация образования',
			year_values JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`ALTER TABLE planning_period_indicators
		  ADD COLUMN IF NOT EXISTS direction TEXT;`,
		`WITH numbered AS (
			SELECT id,
			       ROW_NUMBER() OVER (ORDER BY id ASC) AS rn
			FROM planning_period_indicators
			WHERE COALESCE(NULLIF(BTRIM(direction), ''), '') = ''
			   OR direction NOT IN (
			       'Академическое превосходство и интернационализация образования',
			       'РАЗВИТИЕ НАУКИ И МЕЖДУНАРОДНОГО СОТРУДНИЧЕСТВА',
			       'ЦИФРОВИЗАЦИЯ И МОДЕРНИЗАЦИЯ ИНФРАСТРУКТУРЫ'
			   )
		)
		UPDATE planning_period_indicators AS ppi
		SET direction = CASE
			WHEN ((numbered.rn - 1) % 3) = 0 THEN 'Академическое превосходство и интернационализация образования'
			WHEN ((numbered.rn - 1) % 3) = 1 THEN 'РАЗВИТИЕ НАУКИ И МЕЖДУНАРОДНОГО СОТРУДНИЧЕСТВА'
			ELSE 'ЦИФРОВИЗАЦИЯ И МОДЕРНИЗАЦИЯ ИНФРАСТРУКТУРЫ'
		END
		FROM numbered
		WHERE ppi.id = numbered.id;`,
		`ALTER TABLE planning_period_indicators
		  ALTER COLUMN direction SET DEFAULT 'Академическое превосходство и интернационализация образования';`,
		`UPDATE planning_period_indicators
		 SET direction = 'Академическое превосходство и интернационализация образования'
		 WHERE COALESCE(NULLIF(BTRIM(direction), ''), '') = '';`,
		`ALTER TABLE planning_period_indicators
		  ALTER COLUMN direction SET NOT NULL;`,
		`DO $$
		BEGIN
		  IF NOT EXISTS (
		    SELECT 1
		    FROM pg_constraint
		    WHERE conname = 'planning_period_indicators_direction_check'
		  ) THEN
		    ALTER TABLE planning_period_indicators
		      ADD CONSTRAINT planning_period_indicators_direction_check
		      CHECK (direction IN (
		        'Академическое превосходство и интернационализация образования',
		        'РАЗВИТИЕ НАУКИ И МЕЖДУНАРОДНОГО СОТРУДНИЧЕСТВА',
		        'ЦИФРОВИЗАЦИЯ И МОДЕРНИЗАЦИЯ ИНФРАСТРУКТУРЫ'
		      ));
		  END IF;
		END $$;`,
		`WITH stats AS (
			SELECT COUNT(*) AS total_rows,
			       COUNT(DISTINCT direction) AS distinct_directions,
			       BOOL_AND(direction = 'Академическое превосходство и интернационализация образования') AS only_default_direction
			FROM planning_period_indicators
			WHERE COALESCE(NULLIF(BTRIM(direction), ''), '') <> ''
		),
		numbered AS (
			SELECT id,
			       ROW_NUMBER() OVER (ORDER BY id ASC) AS rn
			FROM planning_period_indicators
		)
		UPDATE planning_period_indicators AS ppi
		SET direction = CASE
			WHEN ((numbered.rn - 1) % 3) = 0 THEN 'Академическое превосходство и интернационализация образования'
			WHEN ((numbered.rn - 1) % 3) = 1 THEN 'РАЗВИТИЕ НАУКИ И МЕЖДУНАРОДНОГО СОТРУДНИЧЕСТВА'
			ELSE 'ЦИФРОВИЗАЦИЯ И МОДЕРНИЗАЦИЯ ИНФРАСТРУКТУРЫ'
		END
		FROM numbered
		WHERE ppi.id = numbered.id
		  AND EXISTS (
		    SELECT 1
		    FROM stats
		    WHERE stats.total_rows > 1
		      AND stats.distinct_directions = 1
		      AND stats.only_default_direction
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
		`CREATE TABLE IF NOT EXISTS indicator_year_targets (
			id BIGSERIAL PRIMARY KEY,
			indicator_id BIGINT NOT NULL REFERENCES planning_period_indicators(id) ON DELETE CASCADE,
			year INT NOT NULL CHECK (year >= 2000 AND year <= 2100),
			planned_value TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (indicator_id, year)
		);`,
		`DO $$
		BEGIN
		  IF EXISTS (
		    SELECT 1
		    FROM information_schema.columns
		    WHERE table_schema = 'public'
		      AND table_name = 'indicator_year_targets'
		      AND column_name = 'planned_value'
		      AND data_type <> 'text'
		  ) THEN
		    ALTER TABLE indicator_year_targets
		      ALTER COLUMN planned_value TYPE TEXT
		      USING TRIM(BOTH FROM planned_value::TEXT);
		  END IF;
		END $$;`,
		`CREATE INDEX IF NOT EXISTS indicator_year_targets_year_idx ON indicator_year_targets (year);`,
		`INSERT INTO indicator_year_targets (indicator_id, year, planned_value, created_at, updated_at)
		SELECT ppi.id,
		       (kv.key)::INT,
		       kv.value,
		       NOW(),
		       NOW()
		FROM planning_period_indicators ppi
		CROSS JOIN LATERAL jsonb_each_text(COALESCE(ppi.year_values, '{}'::jsonb)) kv
		WHERE kv.key ~ '^[0-9]{4}$'
		ON CONFLICT (indicator_id, year)
		DO UPDATE SET planned_value = EXCLUDED.planned_value,
		              updated_at = NOW();`,
		`CREATE TABLE IF NOT EXISTS plan_items (
			id BIGSERIAL PRIMARY KEY,
			indicator_id BIGINT NOT NULL REFERENCES planning_period_indicators(id) ON DELETE CASCADE,
			year INT NOT NULL CHECK (year >= 2000 AND year <= 2100),
			development_indicator TEXT NOT NULL DEFAULT '',
			evaluation_formula TEXT NOT NULL DEFAULT '',
			activities TEXT NOT NULL DEFAULT '',
			execution_deadline TEXT NOT NULL DEFAULT '',
			execution_start_date DATE NULL,
			execution_end_date DATE NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'draft',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (indicator_id, year)
		);`,
		`ALTER TABLE plan_items
		  ADD COLUMN IF NOT EXISTS evaluation_formula TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE plan_items
		  ADD COLUMN IF NOT EXISTS execution_start_date DATE NULL;`,
		`ALTER TABLE plan_items
		  ADD COLUMN IF NOT EXISTS execution_end_date DATE NULL;`,
		`CREATE INDEX IF NOT EXISTS plan_items_year_idx ON plan_items (year);`,
		`CREATE INDEX IF NOT EXISTS plan_items_status_idx ON plan_items (status);`,
		`CREATE TABLE IF NOT EXISTS plan_item_responsibles (
			plan_item_id BIGINT NOT NULL REFERENCES plan_items(id) ON DELETE CASCADE,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (plan_item_id, user_id)
		);`,
		`CREATE INDEX IF NOT EXISTS plan_item_responsibles_user_idx ON plan_item_responsibles (user_id);`,
		`INSERT INTO plan_items (
			indicator_id,
			year,
			development_indicator,
			evaluation_formula,
			activities,
			execution_deadline,
			execution_start_date,
			execution_end_date,
			status,
			created_at,
			updated_at
		)
		SELECT pid.planning_period_indicator_id,
		       pid.year,
		       COALESCE(pid.development_indicator, ''),
		       '',
		       COALESCE(pid.activities, ''),
		       '',
		       NULL,
		       NULL,
		       'draft',
		       COALESCE(pid.created_at, NOW()),
		       COALESCE(pid.updated_at, NOW())
		FROM plan_indicator_details pid
		ON CONFLICT (indicator_id, year)
		DO UPDATE SET
			development_indicator = EXCLUDED.development_indicator,
			evaluation_formula = COALESCE(NULLIF(plan_items.evaluation_formula, ''), EXCLUDED.evaluation_formula),
			activities = EXCLUDED.activities,
			execution_deadline = EXCLUDED.execution_deadline,
			updated_at = NOW();`,
		`INSERT INTO plan_items (
			indicator_id,
			year,
			development_indicator,
			evaluation_formula,
			activities,
			execution_deadline,
			execution_start_date,
			execution_end_date,
			status,
			created_at,
			updated_at
		)
		SELECT pir.planning_period_indicator_id,
		       pir.year,
		       COALESCE(NULLIF(TRIM(pid.development_indicator), ''), ppi.target_indicator, ''),
		       '',
		       COALESCE(pid.activities, ''),
		       '',
		       NULL,
		       NULL,
		       'draft',
		       COALESCE(pir.created_at, NOW()),
		       COALESCE(pir.updated_at, NOW())
		FROM plan_indicator_reports pir
		LEFT JOIN plan_indicator_details pid
		       ON pid.planning_period_indicator_id = pir.planning_period_indicator_id
		      AND pid.year = pir.year
		LEFT JOIN planning_period_indicators ppi
		       ON ppi.id = pir.planning_period_indicator_id
		ON CONFLICT (indicator_id, year)
		DO NOTHING;`,
		`INSERT INTO plan_item_responsibles (plan_item_id, user_id, created_at)
		SELECT pi.id,
		       elem.value::BIGINT,
		       NOW()
		FROM plan_indicator_details pid
		JOIN plan_items pi
		  ON pi.indicator_id = pid.planning_period_indicator_id
		 AND pi.year = pid.year
		CROSS JOIN LATERAL jsonb_array_elements_text(COALESCE(pid.responsible_user_ids, '[]'::jsonb)) elem(value)
		WHERE elem.value ~ '^[0-9]+$'
		ON CONFLICT (plan_item_id, user_id) DO NOTHING;`,
		`INSERT INTO plan_item_responsibles (plan_item_id, user_id, created_at)
		SELECT pi.id,
		       pir.submitted_by,
		       COALESCE(pir.created_at, NOW())
		FROM plan_indicator_reports pir
		JOIN users u
		  ON u.id = pir.submitted_by
		 AND u.role = 'prorector'
		JOIN plan_items pi
		  ON pi.indicator_id = pir.planning_period_indicator_id
		 AND pi.year = pir.year
		ON CONFLICT (plan_item_id, user_id) DO NOTHING;`,
		`CREATE TABLE IF NOT EXISTS report_submissions (
			id BIGSERIAL PRIMARY KEY,
			plan_item_id BIGINT NOT NULL REFERENCES plan_items(id) ON DELETE CASCADE,
			submitted_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			legacy_source_report_id BIGINT NULL,
			report_text TEXT NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			review_note TEXT NOT NULL DEFAULT '',
			approval_formula TEXT NOT NULL DEFAULT '',
			reviewed_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
			submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			reviewed_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CHECK (status IN ('pending', 'completed', 'rejected'))
		);`,
		`ALTER TABLE report_submissions
		  ADD COLUMN IF NOT EXISTS legacy_source_report_id BIGINT NULL;`,
		`WITH mapped AS (
			SELECT rs.id AS submission_id,
			       src.legacy_id
			FROM report_submissions rs
			JOIN plan_items pi
			  ON pi.id = rs.plan_item_id
			JOIN LATERAL (
			  SELECT pir.id AS legacy_id
			  FROM plan_indicator_reports pir
			  WHERE pir.planning_period_indicator_id = pi.indicator_id
			    AND pir.year = pi.year
			    AND pir.submitted_by = rs.submitted_by
			  ORDER BY COALESCE(pir.submitted_at, pir.updated_at, pir.created_at, NOW()) DESC, pir.id DESC
			  LIMIT 1
			) src ON true
			WHERE rs.legacy_source_report_id IS NULL
		)
		UPDATE report_submissions rs
		SET legacy_source_report_id = mapped.legacy_id
		FROM mapped
		WHERE rs.id = mapped.submission_id;`,
		`DO $$
		BEGIN
		  IF EXISTS (
		    SELECT 1
		    FROM pg_constraint
		    WHERE conname = 'report_submissions_plan_item_id_submitted_by_key'
		  ) THEN
		    ALTER TABLE report_submissions
		      DROP CONSTRAINT report_submissions_plan_item_id_submitted_by_key;
		  END IF;
		END $$;`,
		`DROP INDEX IF EXISTS report_submissions_legacy_source_uidx;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS report_submissions_legacy_source_uidx
		 ON report_submissions (legacy_source_report_id);`,
		`CREATE INDEX IF NOT EXISTS report_submissions_status_idx ON report_submissions (status);`,
		`CREATE INDEX IF NOT EXISTS report_submissions_plan_item_idx ON report_submissions (plan_item_id);`,
		`CREATE INDEX IF NOT EXISTS report_submissions_plan_item_submitted_idx ON report_submissions (plan_item_id, submitted_by);`,
		`UPDATE report_submissions
		SET status = LOWER(TRIM(COALESCE(status, '')));`,
		`UPDATE report_submissions
		SET status = 'pending'
		WHERE status = '';`,
		`UPDATE report_submissions
		SET status = 'completed'
		WHERE status = 'approved';`,
		`UPDATE report_submissions
		SET status = 'pending'
		WHERE status NOT IN ('pending', 'completed', 'rejected');`,
		`DO $$
		BEGIN
		  IF EXISTS (
		    SELECT 1
		    FROM pg_constraint
		    WHERE conname = 'report_submissions_status_check'
		  ) THEN
		    ALTER TABLE report_submissions
		      DROP CONSTRAINT report_submissions_status_check;
		  END IF;
		END $$;`,
		`DO $$
		BEGIN
		  IF NOT EXISTS (
		    SELECT 1
		    FROM pg_constraint
		    WHERE conname = 'report_submissions_status_check'
		  ) THEN
		    ALTER TABLE report_submissions
		      ADD CONSTRAINT report_submissions_status_check
		      CHECK (status IN ('pending', 'completed', 'rejected'));
		  END IF;
		END $$;`,
		`INSERT INTO report_submissions (
			plan_item_id,
			submitted_by,
			legacy_source_report_id,
			report_text,
			status,
			review_note,
			approval_formula,
			reviewed_by,
			submitted_at,
			reviewed_at,
			created_at,
			updated_at
		)
		SELECT pi.id,
		       pir.submitted_by,
		       pir.id,
		       COALESCE(pir.report_text, ''),
		       CASE
		           WHEN LOWER(TRIM(COALESCE(pir.status, ''))) IN ('completed', 'approved')
		           THEN 'completed'
		           WHEN LOWER(TRIM(COALESCE(pir.status, ''))) = 'rejected'
		           THEN 'rejected'
		           ELSE 'pending'
		       END,
		       COALESCE(pir.review_note, ''),
		       COALESCE(pir.approval_formula, ''),
		       pir.reviewed_by,
		       COALESCE(pir.submitted_at, NOW()),
		       pir.reviewed_at,
		       COALESCE(pir.created_at, NOW()),
		       COALESCE(pir.updated_at, NOW())
		FROM plan_indicator_reports pir
		JOIN plan_items pi
		  ON pi.indicator_id = pir.planning_period_indicator_id
		 AND pi.year = pir.year
		ON CONFLICT (legacy_source_report_id) DO NOTHING;`,
		`CREATE TABLE IF NOT EXISTS report_files (
			id BIGSERIAL PRIMARY KEY,
			submission_id BIGINT NOT NULL REFERENCES report_submissions(id) ON DELETE CASCADE,
			file_name VARCHAR(255) NOT NULL,
			storage_key TEXT NOT NULL,
			mime_type VARCHAR(255) NOT NULL DEFAULT '',
			file_size BIGINT NOT NULL DEFAULT 0,
			sha256 VARCHAR(64) NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS report_files_submission_idx ON report_files (submission_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS report_files_submission_storage_uindex ON report_files (submission_id, storage_key);`,
		`INSERT INTO report_files (submission_id, file_name, storage_key, mime_type, file_size, sha256, created_at)
		SELECT rs.id,
		       COALESCE(NULLIF(TRIM(prf.file_name), ''), CONCAT('file_', prf.id)),
		       COALESCE(prf.storage_path, ''),
		       '',
		       0,
		       '',
		       COALESCE(prf.created_at, NOW())
		FROM plan_indicator_report_files prf
		JOIN plan_indicator_reports pir
		  ON pir.id = prf.report_id
		JOIN report_submissions rs
		  ON rs.legacy_source_report_id = pir.id
		WHERE COALESCE(prf.storage_path, '') <> ''
		ON CONFLICT (submission_id, storage_key) DO NOTHING;`,
		`INSERT INTO report_files (submission_id, file_name, storage_key, mime_type, file_size, sha256, created_at)
		SELECT rs.id,
		       CASE
		           WHEN COALESCE(NULLIF(TRIM(pir.file_name), ''), '') <> '' THEN pir.file_name
		           ELSE CONCAT('legacy_file_', pir.id)
		       END,
		       pir.file_path,
		       '',
		       0,
		       '',
		       NOW()
		FROM plan_indicator_reports pir
		JOIN report_submissions rs
		  ON rs.legacy_source_report_id = pir.id
		WHERE COALESCE(TRIM(pir.file_path), '') <> ''
		ON CONFLICT (submission_id, storage_key) DO NOTHING;`,
		`CREATE TABLE IF NOT EXISTS report_status_history (
			id BIGSERIAL PRIMARY KEY,
			submission_id BIGINT NOT NULL REFERENCES report_submissions(id) ON DELETE CASCADE,
			from_status VARCHAR(32) NOT NULL DEFAULT '',
			to_status VARCHAR(32) NOT NULL,
			actor_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS report_status_history_submission_idx ON report_status_history (submission_id, created_at DESC);`,
		`INSERT INTO report_status_history (submission_id, from_status, to_status, actor_id, note, created_at)
		SELECT rs.id,
		       '',
		       rs.status,
		       rs.submitted_by,
		       'initial migration',
		       COALESCE(rs.submitted_at, NOW())
		FROM report_submissions rs
		WHERE NOT EXISTS (
			SELECT 1
			FROM report_status_history rsh
			WHERE rsh.submission_id = rs.id
		);`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}
