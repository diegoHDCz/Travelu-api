# Infraestrutura (Terraform)

Provisiona VPC + Aurora Serverless v2 + RDS Proxy + Lambda (API + migrate) +
API Gateway HTTP API + CloudFront + WAF. Ver
`.claude/workload/go-refactor-deploy.md` para o raciocínio por trás de cada
decisão.

## Pré-requisitos

- Terraform >= 1.7
- Credenciais AWS configuradas (`aws configure` ou variáveis de ambiente)
- Os artefatos de deploy já construídos:
  ```bash
  make build-lambda build-migrate
  ```

## Primeira aplicação

```bash
cd deploy/terraform
cp terraform.tfvars.example terraform.tfvars   # edite com valores reais
terraform init
terraform plan
terraform apply
```

Depois do primeiro `apply`, rode a migration uma vez antes de considerar o
ambiente pronto:

```bash
aws lambda invoke \
  --function-name "$(terraform output -raw lambda_migrate_function_name)" \
  --invocation-type RequestResponse \
  /dev/stdout
```

## Atualizações de código (sem mudar infra)

```bash
make build-lambda
aws lambda update-function-code \
  --function-name "$(terraform output -raw lambda_api_function_name)" \
  --zip-file fileb://../../dist/lambda/lambda.zip
```

O workflow de CI/CD (`.github/workflows/deploy.yml`) automatiza isso,
incluindo rodar a migration antes de apontar tráfego para o código novo.

## Backend remoto

Por padrão o state fica local (`terraform.tfstate` neste diretório,
gitignored). Antes de usar isso a sério, troque para um backend S3 +
DynamoDB — o bloco comentado já está em `versions.tf`.
