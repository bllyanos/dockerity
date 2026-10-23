-- User-managed limits; 0 / empty means no limit.
ALTER TABLE apps ADD COLUMN max_ephemeral INTEGER NOT NULL DEFAULT 0;
ALTER TABLE environments ADD COLUMN resource_overrides TEXT NOT NULL DEFAULT '{}';
