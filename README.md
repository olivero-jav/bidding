# bidding

Motor de subastas de un marketplace C2C de coleccionables en Chile. El alcance es
el **motor hasta la adjudicación**: publicación, periodo de pujas con validación
contra el reloj del servidor, y cierre con declaración de ganador. Todo lo
post-adjudicación (pago, despacho, reclamos) queda fuera.

Dos tipos de subasta, una sola entidad con `cap` opcional:

- **Tipo A** — inglesa con fecha de cierre. Cierra al llegar la hora de término.
- **Tipo B** — igual que A, pero si una puja alcanza el `cap` cierra al instante.

## Stack

- **Backend:** Go, monolito modular, arquitectura hexagonal (`domain` → `app` → `port` → `adapter`).
- **Datos:** PostgreSQL como fuente de verdad. Acceso con `pgx` + `sqlc` detrás de un puerto (sin ORM). Migraciones con `golang-migrate`.
- **Concurrencia de pujas:** lock pesimista `SELECT ... FOR UPDATE` sobre la fila de subasta (orden total, sin reintentos) + `lock_timeout` de válvula.
- **Frontend:** Angular 20 (standalone components, signals).

## Estructura

```
backend/
  cmd/api/            punto de entrada del server
  internal/
    auction/          bounded context de subastas
      domain/         entidades e invariantes (Auction, Bid, Money)
      app/            casos de uso
      port/           interfaces (repositorios, clock)
      adapter/        postgres (sqlc) + http (REST)
    platform/         pool de DB, config, clock
  db/
    migrations/       .sql versionados
    queries/          .sql para sqlc
frontend/             app Angular
```

## Cómo correrlo

Requisitos: Go 1.25+, Node 20+, Docker, y las CLIs `migrate` y `sqlc`.

### 1. Base de datos

```bash
cd backend
docker compose up -d          # Postgres en localhost:55432
migrate -path db/migrations -database "postgres://bidding:bidding@localhost:55432/bidding?sslmode=disable" up
```

### 2. Backend

```bash
cd backend
go run ./cmd/api              # API en http://localhost:8080
```

La `DATABASE_URL` por defecto ya apunta a `localhost:55432`; se puede sobrescribir
por variable de entorno, igual que `HTTP_ADDR` (default `:8080`).

### 3. Frontend

```bash
cd frontend
npm install
ng serve                     # app en http://localhost:4200
```

## Tests

```bash
cd backend
go test ./...                # unitarios + test de integración
go test -short ./...         # omite el test que necesita Docker
```

El test **stampede** (`internal/auction/adapter/postgres`) levanta un Postgres
efímero con `testcontainers-go` y dispara N pujas concurrentes idénticas para
comprobar que exactamente una gana y el estado queda consistente. Necesita Docker.

## API

| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/api/auctions` | Crear subasta |
| `GET` | `/api/auctions` | Listar subastas |
| `GET` | `/api/auctions/{id}` | Ver ficha |
| `POST` | `/api/auctions/{id}/bids` | Pujar |

Montos en pesos chilenos (CLP) como enteros; tiempos en UTC (ISO-8601).

## Estado

- **Rebanada 1** — CRUD de subastas full-stack. ✅
- **Rebanada 2** — puja / núcleo de concurrencia. ✅
- Próximo: cierre + adjudicación.
