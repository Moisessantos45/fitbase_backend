-- =============================================================================
-- FITBASE DATABASE SCHEMA FOR POSTGRESQL (psql)
-- Basado en: internal/models/models.go
-- =============================================================================

-- Extensiones recomendadas
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- Si tu servidor cuenta con PostGIS instalado, descomenta la siguiente línea:
CREATE EXTENSION IF NOT EXISTS postgis;

-- -----------------------------------------------------------------------------
-- 1. TABLA: roles
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS roles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- -----------------------------------------------------------------------------
-- 2. TABLA: auths
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS auths (
    id BIGSERIAL PRIMARY KEY,
    role_id BIGINT NOT NULL REFERENCES roles(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    email VARCHAR(150) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    two_factor_secret VARCHAR(26),
    two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    first_session BOOLEAN NOT NULL DEFAULT TRUE,
    full_profile BOOLEAN NOT NULL DEFAULT FALSE,
    email_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    token TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_auths_role_id ON auths(role_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_auths_email ON auths(email);

-- -----------------------------------------------------------------------------
-- 3. TABLA: users (Perfil de Cliente)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    auth_id BIGINT NOT NULL UNIQUE REFERENCES auths(id) ON UPDATE CASCADE ON DELETE CASCADE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_auth_id ON users(auth_id);

-- -----------------------------------------------------------------------------
-- 4. TABLA: owners (Perfil de Dueño de Gimnasios)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS owners (
    id BIGSERIAL PRIMARY KEY,
    auth_id BIGINT NOT NULL UNIQUE REFERENCES auths(id) ON UPDATE CASCADE ON DELETE CASCADE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_owners_auth_id ON owners(auth_id);

-- -----------------------------------------------------------------------------
-- 5. TABLA: gyms (Sedes de Gimnasio)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS gyms (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES owners(id) ON UPDATE CASCADE ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    address VARCHAR(255) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    email VARCHAR(100) NOT NULL,
    latitude NUMERIC(9,6) NOT NULL,
    longitude NUMERIC(9,6) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gyms_owner_id ON gyms(owner_id);

-- -----------------------------------------------------------------------------
-- 6. TABLA: gym_managers (Encargados de Sede)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS gym_managers (
    id BIGSERIAL PRIMARY KEY,
    gym_id BIGINT NOT NULL REFERENCES gyms(id) ON UPDATE CASCADE ON DELETE CASCADE,
    auth_id BIGINT NOT NULL UNIQUE REFERENCES auths(id) ON UPDATE CASCADE ON DELETE CASCADE,
    created_by_auth_id BIGINT NOT NULL REFERENCES auths(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gym_managers_gym_id ON gym_managers(gym_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_gym_managers_auth_id ON gym_managers(auth_id);
CREATE INDEX IF NOT EXISTS idx_gym_managers_created_by_auth_id ON gym_managers(created_by_auth_id);

-- -----------------------------------------------------------------------------
-- 7. TABLA: cash_registers (Caja de Turno)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cash_registers (
    id BIGSERIAL PRIMARY KEY,
    gym_id BIGINT NOT NULL REFERENCES gyms(id) ON UPDATE CASCADE ON DELETE CASCADE,
    auth_id BIGINT NOT NULL REFERENCES auths(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    open_time TIMESTAMPTZ NOT NULL,
    close_time TIMESTAMPTZ DEFAULT NULL,
    open_cash NUMERIC(10,2) NOT NULL,
    close_cash NUMERIC(10,2) DEFAULT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cash_registers_gym_id ON cash_registers(gym_id);
CREATE INDEX IF NOT EXISTS idx_cash_registers_auth_id ON cash_registers(auth_id);

-- -----------------------------------------------------------------------------
-- 8. TABLA: gym_schedules (Horarios de Sede)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS gym_schedules (
    id BIGSERIAL PRIMARY KEY,
    gym_id BIGINT NOT NULL REFERENCES gyms(id) ON UPDATE CASCADE ON DELETE CASCADE,
    day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    open_minutes INT NOT NULL CHECK (open_minutes BETWEEN 0 AND 1440),
    close_minutes INT NOT NULL CHECK (close_minutes BETWEEN 0 AND 1440),
    is_closed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gym_schedules_gym_id ON gym_schedules(gym_id);

-- -----------------------------------------------------------------------------
-- 9. TABLA: payment_methods (Métodos de Pago Catálogo)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payment_methods (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- -----------------------------------------------------------------------------
-- 10. TABLA: plans (Planes de Membresía y Acceso)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS plans (
    id BIGSERIAL PRIMARY KEY,
    gym_id BIGINT NOT NULL REFERENCES gyms(id) ON UPDATE CASCADE ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(255) NOT NULL,
    price NUMERIC(10,2) NOT NULL,
    discount NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    duration INT NOT NULL,
    plan_type VARCHAR(50) NOT NULL,
    max_visits INT DEFAULT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_plans_gym_id ON plans(gym_id);

-- -----------------------------------------------------------------------------
-- 11. TABLA: plan_items (Beneficios o Ítems del Plan)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS plan_items (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES plans(id) ON UPDATE CASCADE ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_plan_items_plan_id ON plan_items(plan_id);

-- -----------------------------------------------------------------------------
-- 12. TABLA: status_subscriptions (Estados de Suscripción Catálogo)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS status_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- -----------------------------------------------------------------------------
-- 13. TABLA: subscriptions (Suscripciones de Clientes)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    plan_id BIGINT NOT NULL REFERENCES plans(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    status_id BIGINT NOT NULL REFERENCES status_subscriptions(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    slug UUID NOT NULL UNIQUE,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    payment_method_id BIGINT NOT NULL REFERENCES payment_methods(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    auto_renew BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_plan_id ON subscriptions(plan_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status_id ON subscriptions(status_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_slug ON subscriptions(slug);

-- -----------------------------------------------------------------------------
-- 14. TABLA: payment_statuses (Estados de Pago Catálogo)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payment_statuses (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- -----------------------------------------------------------------------------
-- 15. TABLA: day_passes (Pases por Día)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS day_passes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    gym_id BIGINT NOT NULL REFERENCES gyms(id) ON UPDATE CASCADE ON DELETE CASCADE,
    plan_id BIGINT NOT NULL REFERENCES plans(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    slug UUID NOT NULL UNIQUE,
    visit_date DATE NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    email VARCHAR(100) NOT NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_day_passes_user_id ON day_passes(user_id);
CREATE INDEX IF NOT EXISTS idx_day_passes_gym_id ON day_passes(gym_id);
CREATE INDEX IF NOT EXISTS idx_day_passes_plan_id ON day_passes(plan_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_day_passes_slug ON day_passes(slug);

-- -----------------------------------------------------------------------------
-- 16. TABLA: payments (Pagos y Cobros)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    gym_id BIGINT NOT NULL REFERENCES gyms(id) ON UPDATE CASCADE ON DELETE CASCADE,
    subscription_id BIGINT REFERENCES subscriptions(id) ON UPDATE CASCADE ON DELETE SET NULL,
    day_pass_id BIGINT REFERENCES day_passes(id) ON UPDATE CASCADE ON DELETE SET NULL,
    cash_register_id BIGINT REFERENCES cash_registers(id) ON UPDATE CASCADE ON DELETE SET NULL,
    payment_method_id BIGINT NOT NULL REFERENCES payment_methods(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    payment_status_id BIGINT NOT NULL REFERENCES payment_statuses(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    amount NUMERIC(10,2) NOT NULL,
    payment_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_payments_gym_id ON payments(gym_id);
CREATE INDEX IF NOT EXISTS idx_payments_subscription_id ON payments(subscription_id);
CREATE INDEX IF NOT EXISTS idx_payments_day_pass_id ON payments(day_pass_id);
CREATE INDEX IF NOT EXISTS idx_payments_cash_register_id ON payments(cash_register_id);
CREATE INDEX IF NOT EXISTS idx_payments_payment_method_id ON payments(payment_method_id);
CREATE INDEX IF NOT EXISTS idx_payments_payment_status_id ON payments(payment_status_id);

-- -----------------------------------------------------------------------------
-- 17. TABLA: access_types (Tipos de Acceso Catálogo)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS access_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- -----------------------------------------------------------------------------
-- 18. TABLA: check_ins (Control de Acceso / Entradas)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS check_ins (
    id BIGSERIAL PRIMARY KEY,
    gym_id BIGINT NOT NULL REFERENCES gyms(id) ON UPDATE CASCADE ON DELETE CASCADE,
    access_type_id BIGINT NOT NULL REFERENCES access_types(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    user_id BIGINT REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    day_pass_id BIGINT REFERENCES day_passes(id) ON UPDATE CASCADE ON DELETE SET NULL,
    subscription_id BIGINT REFERENCES subscriptions(id) ON UPDATE CASCADE ON DELETE SET NULL,
    registered_by_id BIGINT REFERENCES auths(id) ON UPDATE CASCADE ON DELETE SET NULL,
    checked_in_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE gyms
ADD COLUMN IF NOT EXISTS location geography(Point, 4326)
GENERATED ALWAYS AS (
    ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography
) STORED;

CREATE INDEX IF NOT EXISTS idx_gyms_location
ON gyms USING GIST (location);


CREATE INDEX IF NOT EXISTS idx_check_ins_gym_id ON check_ins(gym_id);
CREATE INDEX IF NOT EXISTS idx_check_ins_access_type_id ON check_ins(access_type_id);
CREATE INDEX IF NOT EXISTS idx_check_ins_user_id ON check_ins(user_id);
CREATE INDEX IF NOT EXISTS idx_check_ins_day_pass_id ON check_ins(day_pass_id);
CREATE INDEX IF NOT EXISTS idx_check_ins_subscription_id ON check_ins(subscription_id);
CREATE INDEX IF NOT EXISTS idx_check_ins_registered_by_id ON check_ins(registered_by_id);
CREATE INDEX IF NOT EXISTS idx_check_ins_checked_in_at ON check_ins(checked_in_at);
