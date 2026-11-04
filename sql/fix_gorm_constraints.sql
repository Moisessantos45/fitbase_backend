-- =============================================================================
-- FITBASE: Patch para corregir conflictos de constraints con GORM AutoMigrate
-- Archivo: sql/fix_gorm_constraints.sql
-- =============================================================================
-- Este script corrige las restricciones implícitas creadas por versiones anteriores
-- de 01_schema.sql ("<tabla>_<columna>_key") que causaban el error:
-- "ERROR: constraint uni_auths_email of relation auths does not exist"
-- al intentar iniciar el backend con GORM en PostgreSQL.
-- =============================================================================

BEGIN;

-- 1. Eliminar restricciones UNIQUE de columna implícitas si existen
ALTER TABLE IF EXISTS auths DROP CONSTRAINT IF EXISTS auths_email_key;
ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS users_auth_id_key;
ALTER TABLE IF EXISTS owners DROP CONSTRAINT IF EXISTS owners_auth_id_key;
ALTER TABLE IF EXISTS gym_managers DROP CONSTRAINT IF EXISTS gym_managers_auth_id_key;
ALTER TABLE IF EXISTS subscriptions DROP CONSTRAINT IF EXISTS subscriptions_slug_key;
ALTER TABLE IF EXISTS day_passes DROP CONSTRAINT IF EXISTS day_passes_slug_key;

-- 2. Asegurar que los índices UNIQUE requeridos por GORM (uniqueIndex) existan
CREATE UNIQUE INDEX IF NOT EXISTS idx_auths_email ON auths(email);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_auth_id ON users(auth_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_owners_auth_id ON owners(auth_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_gym_managers_auth_id ON gym_managers(auth_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_slug ON subscriptions(slug);
CREATE UNIQUE INDEX IF NOT EXISTS idx_day_passes_slug ON day_passes(slug);

COMMIT;

