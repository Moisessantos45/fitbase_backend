-- =============================================================================
-- FITBASE: SCRIPT MAESTRO DE INICIALIZACIÓN COMPLETA
-- Ejecutar en PostgreSQL / psql con:
--   psql -h <host> -U <usuario> -d <base_de_datos> -f sql/init.sql
-- =============================================================================

\echo '-> Creando esquema y tablas...'
\i sql/01_schema.sql

\echo '-> Insertando catálogos y datos iniciales (seeds)...'
\i sql/02_seeds.sql

\echo '-> Base de datos inicializada exitosamente.'
