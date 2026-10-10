# Deploy — passo a passo

Guia prático para colocar a travelu-api no ar na AWS (Lambda + Aurora
Serverless v2 + API Gateway + CloudFront/WAF) e configurar o deploy
automático via GitHub Actions. Para o raciocínio por trás de cada decisão de
arquitetura, ver `.claude/workload/go-refactor-deploy.md`; para referência
rápida de comandos Terraform, ver `deploy/terraform/README.md` (este
documento é o passo a passo completo, aquele é o atalho do dia a dia).

## Visão geral da infra

```
cliente → CloudFront (WAF: rate-limit em /api/v1/auth/*)
            → API Gateway HTTP API
                → Lambda "api" (cmd/lambda, na VPC)
                    → RDS Proxy → Aurora Serverless v2

CI/CD → Lambda "migrate" (cmd/migrate, na VPC) → RDS Proxy → Aurora
```

## Pré-requisitos

- Conta AWS com permissão de administrador (só para o bootstrap inicial —
  depois do primeiro apply, o deploy do dia a dia usa uma role com permissão
  mínima, criada pelo próprio Terraform).
- [AWS CLI](https://docs.aws.amazon.com/cli/) configurado (`aws configure`
  ou variáveis `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`/`AWS_SESSION_TOKEN`).
- Terraform >= 1.7 (`brew install hashicorp/tap/terraform`).
- Go 1.27+ (mesma versão do `go.mod`).
- Repositório no GitHub com Actions habilitado (o pipeline já está em
  `.github/workflows/`).

Todos os comandos abaixo assumem que você está na raiz do repositório, exceto
quando indicado.

## Passo 1 — gerar os segredos

```bash
openssl rand -base64 48
```

Guarde esse valor — é o `JWT_SECRET` de produção (precisa ter pelo menos 32
bytes; o app recusa subir se for menor). Não reaproveite o `JWT_SECRET` do
`.env` de desenvolvimento.

## Passo 2 — configurar as variáveis do Terraform

```bash
cd deploy/terraform
cp terraform.tfvars.example terraform.tfvars
```

Edite `terraform.tfvars`:

```hcl
jwt_secret           = "<o valor gerado no passo 1>"
cors_allowed_origins = "https://seu-frontend.example.com"
# alarm_notification_email = "voce@example.com"  # opcional, recebe e-mail quando um alarme disparar
```

`terraform.tfvars` está no `.gitignore` — nunca commite esse arquivo. Se
`github_repo` (em `variables.tf`) não for `diegoHDCz/Travelu-api`, ajuste
também — ele é o que restringe quem pode assumir a role de deploy via OIDC.

## Passo 3 — build dos artefatos de deploy

Volte pra raiz do repo:

```bash
cd ../..
make build-lambda build-migrate
```

Isso gera `dist/lambda/lambda.zip` e `dist/migrate/migrate.zip`, que é o que
o Terraform referencia (`lambda_zip_path`/`migrate_zip_path` em
`variables.tf`). Rode isso antes de todo `terraform apply` que mudar o
código — o Terraform não builda o Go sozinho.

## Passo 4 — provisionar a infraestrutura

```bash
cd deploy/terraform
terraform init
terraform plan
terraform apply
```

Reserve uns 10-15 minutos — a criação do cluster Aurora é a parte mais
lenta. Ao final, anote os outputs (ou rode `terraform output` a qualquer
momento depois):

```bash
terraform output
```

Os outputs importantes: `cloudfront_domain_name` (entrypoint público),
`lambda_api_function_name`, `lambda_migrate_function_name` e
`github_actions_deploy_role_arn` (usado no Passo 7).

## Passo 5 — rodar a migration inicial

As migrations não rodam sozinhas no cold start da Lambda de produção (ver
`.claude/workload/go-refactor-deploy.md`, Fase 1) — é preciso invocar a
Lambda de migration manualmente na primeira vez:

```bash
aws lambda invoke \
  --function-name "$(terraform output -raw lambda_migrate_function_name)" \
  --invocation-type RequestResponse \
  --cli-read-timeout 120 \
  /dev/stdout
```

Espere `{"status":"ok"}` no corpo da resposta e nenhum campo `FunctionError`
no envelope do CLI.

## Passo 6 — testar o endpoint

```bash
curl -s "https://$(terraform output -raw cloudfront_domain_name)/health"
# {"status":"ok","version":"..."}
```

Se vier `503` ou erro de conexão, veja a seção de troubleshooting abaixo
antes de seguir.

## Passo 7 — configurar o deploy automático (GitHub Actions)

No GitHub: `Settings` → `Secrets and variables` → `Actions`.

Em **Secrets**, adicione:

| Nome                  | Valor                                                        |
|-----------------------|---------------------------------------------------------------|
| `AWS_DEPLOY_ROLE_ARN` | `terraform output -raw github_actions_deploy_role_arn`         |

Em **Variables**, adicione:

| Nome            | Valor                                                              |
|-----------------|----------------------------------------------------------------------|
| `API_BASE_URL`  | `terraform output -raw cloudfront_domain_name` (sem `https://`)      |

Essa role usa OIDC federation (sem access key de longa duração salva no
GitHub) e só tem permissão para atualizar o código das duas Lambdas — ver
`deploy/terraform/github_oidc.tf`.

## Passo 8 — deploys seguintes

A partir daqui, o fluxo normal é só:

```bash
git push origin main
```

`.github/workflows/ci.yml` roda build/vet/test/lint; se passar,
`.github/workflows/deploy.yml` builda os artefatos, atualiza a Lambda de
migration, roda a migration, atualiza a Lambda de API e confere `/health`.
Acompanhe em `Actions` no GitHub.

### Deploy manual (sem esperar o CI)

```bash
make build-lambda
aws lambda update-function-code \
  --function-name "$(terraform output -raw lambda_api_function_name)" \
  --zip-file fileb://dist/lambda/lambda.zip
```

### Mudou algo na infra (não só no código Go)

```bash
cd deploy/terraform
terraform plan
terraform apply
```

## Rollback

Este setup não mantém versões anteriores do zip automaticamente. Para
reverter um deploy ruim:

```bash
git checkout <commit-anterior-bom> -- .
make build-lambda
aws lambda update-function-code \
  --function-name "$(terraform output -raw lambda_api_function_name)" \
  --zip-file fileb://dist/lambda/lambda.zip
```

Se o commit ruim incluiu uma migration destrutiva, o rollback de código
sozinho não resolve — é preciso uma migration de reversão (`-- +goose Down`
correspondente) aplicada manualmente via `cmd/migrate`.

## Troubleshooting comum

- **`/health` retorna 503**: Lambda não conseguiu conectar no banco. Confira
  os security groups (`deploy/terraform/vpc.tf`) e se o RDS Proxy está
  `AVAILABLE` (`aws rds describe-db-proxies`).
- **Timeout na primeira invocação depois do apply**: cold start com ENI de
  VPC pode levar alguns segundos a mais na primeira chamada; invocações
  seguintes (warm) são rápidas.
- **`terraform apply` falha no `aws_iam_openid_connect_provider`**: geralmente
  já existe um provider OIDC do GitHub Actions na conta (de outro projeto) —
  rode `aws iam list-open-id-connect-providers` e, se achar um pra
  `token.actions.githubusercontent.com`, importe-o
  (`terraform import aws_iam_openid_connect_provider.github_actions <arn>`)
  em vez de deixar o Terraform tentar criar outro.
- **Migration trava além do timeout**: `cmd/migrate` tem timeout de 60s
  internamente e a Lambda de 120s (`deploy/terraform/lambda.tf`); migrations
  muito grandes podem precisar de ambos aumentados.

## Desligar tudo

```bash
cd deploy/terraform
terraform destroy
```

Cobrança para ter em mente enquanto estiver no ar: Aurora Serverless v2 cobra
por ACU mesmo no mínimo (0.5 ACU parado, não é gratuito como seria uma VM
Always Free), e CloudFront/WAF têm custo pequeno por request. Lambda e API
Gateway são pay-per-use e ficam perto de zero sem tráfego.
