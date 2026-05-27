# Pipefy Integration API

API REST em Go para gerenciamento de clientes e simulação de integração com o serviço Pipefy.

---

## Visão Geral

A aplicação expõe dois fluxos principais:

1. **`POST /clientes`** — Cria um novo cliente no banco de dados com status `Aguardando Análise` e simula o envio de um card de criação ao Pipefy via GraphQL (`createCard`).

2. **`POST /webhooks/pipefy/card-updated`** — Simula o recebimento de um webhook do Pipefy quando um card é atualizado. Aplica a regra de prioridade baseada no patrimônio do cliente, garante **idempotência** por `event_id` e simula o envio da mutation GraphQL de atualização ao Pipefy (`updateFieldsValues`).

---

## Tecnologias

- **Go 1.26+** com `net/http` da stdlib
- **PostgreSQL 18** via Docker
- **pgx/v5** - driver PostgreSQL nativo
- **go-playground/validator v10** — validação de structs
- **google/uuid** - geração de UUIDs v7 para IDs dos clientes
- **Docker / Docker Compose** - ambiente containerizado
- **testcontainers/testcontainers-go** - containers de teste para testes de integração e e2e

---

## Pré-requisitos

- [Docker](https://docs.docker.com/get-docker/) e [Docker Compose](https://docs.docker.com/compose/install/) instalados

---

## Configuração e Execução

### 1. Copie o arquivo de variáveis de ambiente

```bash
cp .env.example .env
```

Edite o `.env` com os valores desejados:

```env
HTTP_PORT=8080

POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=clients

# DSN usado pela aplicação para conectar ao postgres dentro do Docker
DSN=postgres://postgres:postgres@postgres:5432/clients
```

### 2. Suba os containers

```bash
docker compose up --build
```

O Docker Compose irá:
- Subir o **PostgreSQL** com healthcheck
- Executar automaticamente os scripts de migração em `./sql/` na ordem numérica
- Subir a **aplicação** após o banco estar pronto

A API estará disponível em `http://localhost:8080`.

### 3. Parar os containers

```bash
docker compose down
```

---

## Exemplos de Requisição (cURL)

### Criar cliente

```bash
curl -s -X POST http://localhost:8080/clientes \
  -H "Content-Type: application/json" \
  -d '{"cliente_nome": "Filipe Costa","cliente_email": "filipe.costa@example.com","tipo_solicitacao": "Atualização cadastral","valor_patrimonio": 250000}'
```

**Resposta esperada (`202 Accepted`):**
```json
{"client_id":"019e600f-6b01-7efc-b853-55daa280466f"}
```

---

### Processar webhook (atualização de card)

```bash
curl -s -X POST http://localhost:8080/webhooks/pipefy/card-updated \
  -H "Content-Type: application/json" \
  -d '{"event_id": "evt_001","card_id": "card_001","cliente_email": "filipe.costa@example.com","timestamp": "2026-05-25T15:00:00-03:00"}'
```

**Resposta esperada (`200 OK`):**
```json
{"processed":"evt_001"}
```

---

## Testes

O projeto tem testes automáticos, com testes unitários com mock da Store para execução em formato de table tests.
Também possui testes de integração dos handlers da aplicação e teste E2E, que valida todo o ciclo.
Os testes cobrem:

1. **Criação de cliente** com payload válido e cliente duplicado
2. **Processamento do webhook** com aplicação correta de regra de prioriodade
3. **Idempotência**: bloqueio ao reprocessar um `event_id` duplicado

### Executar os testes

Os testes de integração e E2E utilizam o **testcontainers-go**, que sobe um container PostgreSQL real durante a execução. Por isso, **o Docker precisa estar rodando** para executar a suite completa:

```bash
go test -v ./...
```

Caso o Docker não esteja disponível, use a flag `-short` para rodar apenas os testes unitários (os testes de integração e E2E são automaticamente pulados):

```bash
go test -v -short ./...
```

---

## Visão de Produção na AWS

Para um projeto de implantação na AWS, teríamos um API Gateway, que ficaria na frente da aplicação gerenciando autenticação e roteamento, despachando cada requisição para uma Lambda dedicada por endpoint. As Lambdas escalam automaticamente com a carga, sem necessidade de gerenciar servidores. O banco continuaria PostgreSQL no RDS com Multi-AZ para alta disponibilidade, e o RDS Proxy seria colocado no meio para evitar que múltiplas instâncias Lambda estourem o pool de conexões do banco.
