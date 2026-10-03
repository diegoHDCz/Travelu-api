# travelu-api

API REST monolítica em Go, organizada por domínio, usando apenas a stdlib `net/http`
para roteamento/HTTP e um conjunto pequeno e explícito de bibliotecas para banco,
autenticação e validação. Sem ORM, sem framework web, sem query builder.

## Stack

- HTTP: `net/http` (roteamento nativo do Go 1.22+, `mux.HandleFunc("GET /users/{id}", ...)`)
- Banco: MySQL 8.4+ via `go-sql-driver/mysql` + `jmoiron/sqlx`, SQL escrito à mão
- Migrations: `pressly/goose` com os `.sql` embutidos no binário via `go:embed`
- Autenticação: JWT HS256 (`golang-jwt/jwt/v5`) + refresh token opaco com rotação
- Senhas: `bcrypt`
- Config: variáveis de ambiente via `caarlos0/env`
- Validação: `go-playground/validator`
- Logs: `log/slog` (JSON em produção, texto em desenvolvimento)

## Estrutura

```
cmd/api/main.go              fiação manual das dependências, graceful shutdown
internal/
  config/                    carregamento de configuração via env vars
  platform/
    database/                pool de conexões sqlx + helper de transação
    migrations/               migrations embutidas (go:embed) + execução no startup
    httpx/                    JSON helpers, erro padrão, middlewares genéricos
  user/                       entidade, repositório, service e regras de negócio
  auth/                       JWT, refresh tokens, handlers de auth e GET /users/me
```

> **Nota de arquitetura:** `GET /api/v1/users/me` é implementado no pacote `auth`,
> não em `user`. Ele precisa do ID do usuário autenticado, que `auth.RequireAuth`
> coloca no contexto via `auth.UserIDFrom`. Como `auth` já depende de `user` (para
> criar/consultar usuários), colocar esse handler em `user` criaria um import cycle
> (`user` → `auth` → `user`). `user` permanece sem nenhuma dependência de `auth`.

## Rodando localmente

Pré-requisitos: Go 1.22+ e Docker (apenas para o MySQL de desenvolvimento — a
aplicação em si não roda em container).

```bash
cp .env.example .env
# edite o .env se quiser, principalmente o JWT_SECRET

make db-up        # sobe um MySQL 8.4 local em localhost:3306
set -a && source .env && set +a
make run          # roda cmd/api; aplica as migrations automaticamente no startup
```

O servidor sobe em `http://localhost:8080` (em desenvolvimento, escutando em todas
as interfaces; em produção, só em `127.0.0.1`, atrás do Caddy).

Para derrubar o banco de desenvolvimento: `make db-down`.

## Variáveis de ambiente

Veja `.env.example` para a lista completa com comentários. Resumo:

| Variável                | Obrigatória | Descrição                                                              |
|--------------------------|:-----------:|--------------------------------------------------------------------------|
| `APP_ENV`                |     não      | `development` (default) ou `production`                                |
| `PORT`                   |     não      | porta HTTP interna (default `8080`)                                     |
| `DATABASE_DSN`           |     sim      | DSN do MySQL (`parseTime=true&loc=UTC&charset=utf8mb4`, +`tls=preferred` em produção) |
| `JWT_SECRET`             |     sim      | segredo HS256; falha no startup se tiver menos de 32 bytes              |
| `ACCESS_TOKEN_TTL`       |     não      | validade do access token (default `15m`)                                |
| `REFRESH_TOKEN_TTL`      |     não      | validade do refresh token (default `720h`, 30 dias)                     |
| `CORS_ALLOWED_ORIGINS`   |     não      | origens permitidas, separadas por vírgula; vazio desabilita CORS        |
| `TRUST_PROXY`            |     não      | `true` só quando atrás do Caddy; habilita leitura de `X-Forwarded-For`  |
| `LOG_LEVEL`              |     não      | `debug`, `info` (default), `warn` ou `error`                            |

## Endpoints

### `GET /health`

Usado pelo processo de deploy para decidir um rollback: faz `PingContext` no banco
e responde `503` se falhar.

```bash
curl -s http://localhost:8080/health
# {"status":"ok","version":"dev"}
```

### `POST /api/v1/auth/register`

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Diego","email":"diego@example.com","password":"supersecret"}'
```

### `POST /api/v1/auth/login`

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"diego@example.com","password":"supersecret"}'
# {"access_token":"...","refresh_token":"...","expires_in":900}
```

Por segurança, a mensagem de erro é genérica tanto para e-mail inexistente quanto
para senha errada — não é possível descobrir se um e-mail está cadastrado.

### `POST /api/v1/auth/refresh`

Troca um refresh token válido por um novo par de tokens. O token usado é revogado
na hora (rotação): reutilizá-lo depois disso é tratado como possível roubo e revoga
**todos** os refresh tokens daquele usuário.

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token recebido no login>"}'
```

### `POST /api/v1/auth/logout`

Revoga um refresh token específico. Idempotente — chamar de novo com o mesmo
token (já revogado) não é erro.

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}'
```

### `GET /api/v1/users/me`

Rota protegida: exige `Authorization: Bearer <access_token>`.

```bash
curl -s http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <access_token recebido no login>"
```

## Fluxo de autenticação

1. `register` cria o usuário (senha com `bcrypt`, nunca armazenada em texto puro).
2. `login` verifica e-mail/senha e devolve um `access_token` (JWT HS256, 15 min,
   claims `sub`/`exp`/`iat`/`jti`) e um `refresh_token` opaco (32 bytes aleatórios,
   codificados em base64url). Só o hash SHA-256 do refresh token é persistido.
3. O `access_token` é enviado em `Authorization: Bearer` nas rotas protegidas;
   `auth.RequireAuth` valida a assinatura (rejeitando qualquer algoritmo que não
   seja HS256) e expõe o ID do usuário via `auth.UserIDFrom(ctx)`.
4. Quando o `access_token` expira, o cliente chama `refresh` com o `refresh_token`.
   O token antigo é revogado e um novo par é emitido (rotação a cada uso).
5. `logout` revoga o `refresh_token` informado, encerrando aquela sessão.

As rotas `register` e `login` têm um rate limit simples em memória por IP, para
dificultar força bruta.

## Testes

```bash
make test              # testes unitários e de handlers (go test -race ./...)
make test-integration  # sobe o MySQL de dev e roda os mesmos testes com TEST_DATABASE_DSN
```

Os testes de repositório (`internal/user` e `internal/auth`) usam `t.Skip` quando
`TEST_DATABASE_DSN` não está definida, então `make test` funciona sem banco algum.

## Build e deploy

```bash
make build   # CGO_ENABLED=0 GOOS=linux GOARCH=arm64, saída em dist/api
```

Em produção: binário estático rodando via `systemd` numa VM Linux ARM64, atrás do
Caddy (proxy reverso com HTTPS). A aplicação escuta apenas em `127.0.0.1` quando
`APP_ENV=production`, já que só o Caddy deve acessá-la. `GET /health` é o endpoint
usado pelo pipeline de deploy para decidir um rollback.
