-- =============================================================================
-- FITBASE SEED DATA FOR POSTGRESQL (psql)
-- Catálogos base del sistema definidos en internal/models/models.go
-- =============================================================================


-- -----------------------------------------------------------------------------
-- 1. ROLES
-- -----------------------------------------------------------------------------
INSERT INTO roles (id, name, created_at, updated_at) VALUES
    (1, 'Admin', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'Owner', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (3, 'GymManager', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (4, 'User', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    updated_at = CURRENT_TIMESTAMP;

-- -----------------------------------------------------------------------------
-- 2. MÉTODOS DE PAGO (payment_methods)
-- -----------------------------------------------------------------------------
INSERT INTO payment_methods (id, name, active, created_at, updated_at) VALUES
    (1, 'Efectivo', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'Tarjeta de crédito', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (3, 'Tarjeta de débito', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (4, 'Transferencia bancaria', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (5, 'PayPal', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (6, 'Criptomonedas', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    active = EXCLUDED.active, 
    updated_at = CURRENT_TIMESTAMP;

-- -----------------------------------------------------------------------------
-- 3. ESTADOS DE SUSCRIPCIÓN (status_subscriptions)
-- -----------------------------------------------------------------------------
INSERT INTO status_subscriptions (id, name, description, created_at, updated_at) VALUES
    (1, 'Activo', 'La suscripción está activa y en uso.', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'Inactivo', 'La suscripción ha sido desactivada o cancelada.', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (3, 'Pendiente', 'La suscripción está pendiente de activación o pago.', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (4, 'Expirado', 'La suscripción ha expirado y ya no es válida.', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    description = EXCLUDED.description, 
    updated_at = CURRENT_TIMESTAMP;

-- -----------------------------------------------------------------------------
-- 4. ESTADOS DE PAGO (payment_statuses)
-- -----------------------------------------------------------------------------
INSERT INTO payment_statuses (id, name, active, created_at, updated_at) VALUES
    (1, 'Pendiente', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'Pagado', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (3, 'Cancelado', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (4, 'Reembolsado', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    active = EXCLUDED.active, 
    updated_at = CURRENT_TIMESTAMP;

-- -----------------------------------------------------------------------------
-- 5. TIPOS DE ACCESO (access_types)
-- -----------------------------------------------------------------------------
INSERT INTO access_types (id, name, active, created_at, updated_at) VALUES
    (1, 'Membresía', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, 'Día de Pase', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (3, 'Invitado', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    active = EXCLUDED.active, 
    updated_at = CURRENT_TIMESTAMP;

-- -----------------------------------------------------------------------------
-- 6. SINCRONIZACIÓN DE SECUENCIAS
-- -----------------------------------------------------------------------------
SELECT setval('roles_id_seq', COALESCE((SELECT MAX(id) FROM roles), 1));
SELECT setval('payment_methods_id_seq', COALESCE((SELECT MAX(id) FROM payment_methods), 1));
SELECT setval('status_subscriptions_id_seq', COALESCE((SELECT MAX(id) FROM status_subscriptions), 1));
SELECT setval('payment_statuses_id_seq', COALESCE((SELECT MAX(id) FROM payment_statuses), 1));
SELECT setval('access_types_id_seq', COALESCE((SELECT MAX(id) FROM access_types), 1));
