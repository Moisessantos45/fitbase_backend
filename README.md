# Fitbase API

Backend REST en **Go** para la gestión integral de gimnasios y membresías: sedes, planes, suscripciones, pases diarios, pagos, control de acceso (QR) y analítica.

- Roles principales: **Dueño (Owner)**, **Encargado (GymManager)** y **Cliente (User)**
- Auth con **PASETO v2**, refresh tokens y **2FA/TOTP**
- **PostgreSQL** (GORM + PostGIS), **Redis** (caché, rate limit, sesiones), **Stripe** y correo transaccional

---

## Stack

| Componente | Tecnología |
| :--- | :--- |
| Lenguaje | Go 1.26+ |
| HTTP | Gin Gonic |
| BD | PostgreSQL + GORM (`pgx/v5`) + PostGIS |
| Caché / infra | Redis (`go-redis/v9`) |
| Tokens | PASETO v2 (`o1egl/paseto`) |
| MFA | TOTP (`xlzd/gotp`) |
| Pagos | Stripe |
| Correo | SMTP / Resend |
| Seguridad HTTP | gin-helmet, CORS, GZIP, rate limiting |

---

## Estructura del proyecto

```text
fitbase/
├── main.go                    # Entry point: .env, DB, Redis, Gin, rutas, graceful shutdown
├── go.mod / go.sum
├── .env                       # Variables locales (no commitear; ver .gitignore)
│
├── config/
│   ├── db/
│   │   ├── connection.go      # Pool GORM → PostgreSQL
│   │   ├── initialize.go      # AutoMigrate (dev) + seed
│   │   └── seed.go            # Catálogos iniciales (roles, métodos de pago, etc.)
│   ├── email.go               # Config SMTP
│   └── redis.go               # Cliente Redis
│
├── sql/                       # Scripts SQL opcionales
│   ├── init.sql               # schema + seeds
│   ├── 01_schema.sql
│   └── 02_seeds.sql
│
├── docs/                      # Documentación técnica (ver sección Documentación)
│   ├── project_structure.md
│   ├── plan_implementacion.md
│   ├── seguridad_api.md
│   ├── owasp_top_10_2025.md
│   ├── features/<feature>/README.md
│   └── tasks/ ...
│
└── internal/
    ├── features/              # Vertical slices (una por módulo de negocio)
    │   ├── auth/              # domain.go | handler.go | usecase.go | repository.go
    │   ├── user/
    │   ├── owner/
    │   ├── gym/
    │   ├── plan/
    │   ├── manager/
    │   ├── daypass/
    │   ├── subscription/
    │   ├── checkin/
    │   ├── payment/
    │   ├── analytics/
    │   └── storage/
    ├── models/                # Modelos GORM, paginación, serializers
    ├── routes/                # deps.go + registro de rutas por módulo
    └── shared/
        ├── errorsx/           # Errores de negocio
        ├── helper/            # BindJSON, RespondError, traducción de errores BD
        ├── middleware/        # Auth, rate limit, request guard, recovery, etc.
        ├── service/           # Servicios genéricos con Redis
        ├── templates/         # HTML de correos (embed)
        └── utils/             # JWT/PASETO, bcrypt, mail, URLs, paginación, etc.
```

Flujo por request: **Handler → UseCase → Repository → PostgreSQL / Redis**.

---

## Requisitos

- Go **1.26+**
- PostgreSQL **con extensión PostGIS**
- Redis
- (Opcional) Stripe y proveedor SMTP para pagos y correos

---

## Puesta en marcha

### 1. Clonar e instalar dependencias

```bash
git clone <repo>
cd fitbase
go mod download
```

### 2. Base de datos

Crear la base de datos y habilitar PostGIS. Opciones:

**A) Migración automática al arrancar (desarrollo)**  
Con `GO_ENV` distinto de `prod`, `main.go` ejecuta `AutoMigrate` + `SeedDatabase()` en el arranque.

**B) Scripts SQL manuales**

```bash
psql -h <host> -U <usuario> -d <base> -f sql/init.sql
# o por partes:
psql -h <host> -U <usuario> -d <base> -f sql/01_schema.sql
psql -h <host> -U <usuario> -d <base> -f sql/02_seeds.sql
```

> En `GO_ENV=prod` se **omiten** migraciones automáticas para proteger el esquema.

### 3. Variables de entorno

Crear un archivo `.env` en la raíz del proyecto. El servidor lo carga con `godotenv`; si no existe, usa las variables del sistema.

**Solo nombres de variables** (rellena cada valor en tu `.env`; no commitees el archivo):

#### Entorno

| Variable | Uso |
| :--- | :--- |
| `GO_ENV` | `prod` activa modo producción (bind en `127.0.0.1`, sin migraciones, CORS restringido, trusted proxies). Cualquier otro valor = desarrollo. |
| `HOST_URL_DEV` | Origen frontend en desarrollo (CORS) |
| `HOST_URL_DEV_V2` | Segundo origen frontend en desarrollo (CORS) |
| `HOST_URL_PROD` | Origen frontend producción (CORS); también fallback de URLs en `routes.Init` |
| `HOST_URL_PROD_WWW` | Variante `www` del frontend en producción (CORS) |
| `HOST_API_PROD` | Base de la API en producción (URLs públicas) |
| `HOST_API_PROD_WWW` | Variante `www` de la API en producción (CORS) |
| `HOST_API_DEV` | Base de la API en desarrollo (URLs de emails/tickets) |
| `PUBLIC_API_BASE` | Base pública alternativa para URLs firmadas |
| `HOST_PYTHON_API_URL` | Origen de API Python asociada (CORS) |
| `COOKIE_DOMAIN` | Dominio de cookies `refresh_token` / `pre_auth_session` |

#### PostgreSQL

| Variable | Uso |
| :--- | :--- |
| `HOST` | Host de PostgreSQL |
| `PORT` | Puerto de PostgreSQL |
| `DBUSER` | Usuario de PostgreSQL |
| `PASSWORD` | Contraseña de PostgreSQL |
| `DBNAME` | Nombre de la base de datos |

#### Redis

| Variable | Uso |
| :--- | :--- |
| `REDIS_HOST` | Host Redis (default `localhost`) |
| `REDIS_PORT` | Puerto Redis (default `6379`) |
| `REDIS_PASSWORD` | Contraseña Redis (opcional) |
| `REDIS_DB` | Índice de base Redis (default `0`) |

#### Seguridad / sesión

| Variable | Uso |
| :--- | :--- |
| `SECRET_KEY_JWT` | Secreto para firmar/validar tokens PASETO |

#### Correo (SMTP)

| Variable | Uso |
| :--- | :--- |
| `SMTP_HOST` | Host SMTP |
| `SMTP_PORT` | Puerto SMTP |
| `SMTP_USER` | Usuario SMTP |
| `API_KEY_SMTP` | Clave/API key SMTP |
| `API_KEY_SMTP_KEY` | Clave SMTP adicional (dashboard) |
| `ADMIN_EMAIL` | Correo administrativo de contacto |

#### Pagos (Stripe)

| Variable | Uso |
| :--- | :--- |
| `STRIPE_SECRET_KEY` | Clave secreta de Stripe |
| `STRIPE_WEBHOOK_SECRET` | Secreto de verificación de webhooks Stripe |

### 4. Ejecutar

```bash
# Desarrollo
go run .

# O compilar
go build -o fitbase .
./fitbase
```

El servidor escucha en **`:4109`** (en `GO_ENV=prod` solo en `127.0.0.1:4109`, detrás de proxy inverso).

### 5. Health checks

```bash
curl http://localhost:4109/health
# {"status":"ok"}

curl http://localhost:4109/api/test
# {"message":"OK"}
```

---

## Cómo usar la API

### Base URL

```text
http://localhost:4109/api/v1
```

### Autenticación

| Flujo | Endpoint | Notas |
| :--- | :--- | :--- |
| Registro cliente | `POST /api/v1/auth/register` | Confirma email |
| Registro dueño | `POST /api/v1/auth/register-owner` | Rol Owner |
| Login web | `POST /api/v1/auth/login` | Access token + cookie `refresh_token` (HttpOnly) |
| Login móvil | `POST /api/v1/auth/login-mobile` | Tokens en JSON |
| Refresh web | `POST /api/v1/auth/refresh-token` | Cookie o Bearer |
| Refresh móvil | `POST /api/v1/auth/refresh-token-mobile` | `Authorization: Bearer <refresh_token>` |
| Logout | `POST /api/v1/auth/logout` / `logout-mobile` | Revoca sesión en Redis |
| Confirmar cuenta | `GET /api/v1/auth/confirm-account` | |
| Forgot / reset | `POST /api/v1/auth/forgot-password`, `PATCH /api/v1/auth/reset-password` | |
| Sesión actual | `GET /api/v1/auth/session` | |

**Endpoints protegidos**: cabecera

```http
Authorization: Bearer <access_token>
```

Los identificadores de usuario se obtienen del token (no del body) para evitar IDOR.

### Módulos principales (prefijo `/api/v1`)

| Módulo | Ejemplos de rutas | Roles típicos |
| :--- | :--- | :--- |
| **auth** | `/auth/*` | Público / sesión |
| **user** | `GET/PUT /users/me`, avatar | Cliente |
| **owner** | `POST /owner/complete-profile`, `GET/PUT /owner/me` | Dueño |
| **gym** | `GET /gyms` (público), `POST /gyms`, schedules, photos | Público / Dueño |
| **plan** | `GET /gyms/:id/plans`, `POST /plans`, items | Público / Dueño |
| **manager** | `POST /managers`, `GET /gyms/managers`, caja open/close/current | Dueño / Encargado |
| **daypass** | `POST /daypass/purchase`, `register-guest`, `guest`, ticket QR | Cliente / Staff / Público |
| **subscription** | `POST /subscriptions/subscribe`, renew, cancel, ticket | Cliente / Staff |
| **checkin** | `POST /checkin/scan`, `GET /checkin/my-history`, `GET /gyms/:id/checkins` | Encargado / Dueño / Cliente |
| **payment** | `POST /payments/cash`, Stripe checkout/webhook, revenue | Staff / Público (webhook) |
| **analytics** | shifts, attendance, revenue, clients, dashboard-summary | Dueño |
| **storage** | `GET /storage/public/:slug`, CRUD protegido | Público / autenticado |

Documentación detallada por módulo (DTOs, casos de uso, contratos JSON):  
`docs/features/<nombre>/README.md`

### Ejemplo mínimo

```bash
# Login (web)
curl -s -X POST http://localhost:4109/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"...","password":"..."}'

# Consultar perfil (con token)
curl -s http://localhost:4109/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"
```

---

## Comportamiento por entorno (`GO_ENV`)

| Aspecto | Desarrollo | Producción (`GO_ENV=prod`) |
| :--- | :--- | :--- |
| Bind | `:4109` | `127.0.0.1:4109` |
| Migraciones | AutoMigrate + seed | Omitidas |
| CORS | Incluye orígenes dev (`HOST_URL_DEV`, `HOST_URL_DEV_V2`, etc.) | Solo orígenes prod configurados |
| Trusted proxies | `nil` | `127.0.0.1`, `::1` |

Rate limit global vía Redis: ~5 req/s (burst 50).  
Sólo se aceptan `application/json` y `multipart/form-data` (RequestGuard).

---

## Documentación (`docs/`)

| Documento | Contenido |
| :--- | :--- |
| `docs/project_structure.md` | Arquitectura, capas y guía de nueva feature |
| `docs/plan_implementacion.md` | Roadmap por fases y estándar de docs por feature |
| `docs/reglas.txt` | Reglas del proyecto |
| `docs/recomendaciones_gorm.md` | Convenciones GORM |
| `docs/seguridad_api.md` | Checklist de seguridad API |
| `docs/owasp_top_10_2025.md` | OWASP Top 10 y mitigaciones |
| `docs/cambios_seguridad_hallazgos.md` | Hallazgos y cambios de seguridad |
| `docs/implements/` | Implementaciones p. ej. sesiones móvil/web |
| `docs/features/*/README.md` | Contratos y ADR por módulo |
| `docs/tasks/` | Tareas por rol (dueño, encargado, cliente) |
| `docs/tasks_team/` | Tareas de equipo por flujo |
| `docs/ui_desing.md` | Notas de UI |

---

## Tests y calidad

```bash
go vet ./...
go build ./...
```

(Agregar scripts de lint/test al repo cuando se definan.)

---

## Seguridad (resumen)

- PASETO v2 + refresh rotation + sesiones en Redis  
- 2FA/TOTP opcional  
- Bcrypt para contraseñas  
- Helmet, CORS restringido, GZIP, rate limit, límites de body/JSON  
- Errores 5xx genéricos al cliente; detalle solo en logs  
- Validación de payloads (`BindJSON` + `RespondError`)

Detalle completo: `docs/seguridad_api.md`.

---

## Licencia

Ver [LICENSE](./LICENSE).
