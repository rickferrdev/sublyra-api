# sublyra-api — documentação técnica

`sublyra-api` é um projeto de estudo sobre fluxos confiáveis e assíncronos de inscrição. Ele oferece uma API HTTP para inscrição e cancelamento de newsletters, persiste o estado das inscrições no MongoDB e registra o evento de integração correspondente usando o padrão **Transactional Outbox**.

> Escopo atual: o lado de escrita transacional está implementado. O relay da outbox, o publisher/consumer do RabbitMQ e o adapter de e-mail do Resend estão planejados, mas ainda não foram implementados.

To read this documentation in English, see [`README.md`](README.md).

## Arquitetura

O código segue uma estrutura inspirada em Ports and Adapters e usa Uber Fx para injeção de dependências e gerenciamento do ciclo de vida.

```text
cmd/api/                         ponto de entrada da aplicação
internal/
├── config/                      configuração do ambiente
├── core/
│   ├── domain/                  modelos de inscrição e outbox
│   ├── ports/                   contrato de erros da aplicação
│   └── services/                casos de uso de inscrição
├── inbound/http/rest/           controllers e middlewares Fiber
├── outbound/mongodb/            repositórios e schemas MongoDB
├── infra/                       servidor HTTP, logger e cliente MongoDB
└── platform/                    utilitários de JWT e validação
```

O fluxo de dependências é:

```text
requisição HTTP → controller Fiber → serviço de inscrição → repositório MongoDB
                                                           ├─ subscriptions
                                                           └─ outbox
```

Principais tecnologias: Go 1.26.3, Fiber v3, MongoDB Go Driver v2, Uber Fx e JWT.

## Transactional Outbox

O problema estudado aqui é o problema da escrita dupla. Uma solicitação de inscrição precisa atualizar o estado de negócio e, posteriormente, disparar um efeito externo, como o envio de um e-mail. Gravar no MongoDB e chamar diretamente um provedor de e-mail são duas operações independentes: uma pode ter sucesso enquanto a outra falha.

A implementação atual grava os dois documentos dentro de uma única transação do MongoDB:

1. O serviço cria ou atualiza um documento em `subscriptions.subscriptions`.
2. Na mesma transação, adiciona um evento em `subscriptions.outbox`.
3. O MongoDB confirma as duas operações ou desfaz ambas.
4. Um relay futuro lerá os eventos pendentes da outbox e os publicará no RabbitMQ.
5. Um consumer futuro usará o Resend para enviar o e-mail de confirmação ou cancelamento.

Os métodos transacionais são `InsertWithOutbox`, `RenewConfirmationWithOutbox` e `RenewUnsubscribedWithOutbox`, localizados no repositório MongoDB de inscrições.

### Requisito importante do MongoDB

Transações do MongoDB envolvendo múltiplos documentos exigem um replica set ou cluster fragmentado. Uma instância standalone não é suficiente. O ambiente local deve apontar `MONGO_URI` para uma implantação configurada como replica set.

### Collections

`subscriptions.subscriptions` armazena o agregado:

```json
{
  "_id": "ObjectId",
  "email": "person@example.com",
  "status": "pending | subscribed | unsubscribed",
  "confirmation_token": "JWT opcional",
  "unsubscribe_token": "JWT opcional",
  "subscribed_at": "datetime opcional",
  "unsubscribed_at": "datetime opcional",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

O campo `email` possui um índice único.

`subscriptions.outbox` armazena os eventos de integração:

```json
{
  "_id": "ObjectId",
  "aggregate_id": "ObjectId da inscrição",
  "email": "person@example.com",
  "event": "outbox_subscription_confirmation_requested",
  "attempts": 0,
  "payload": {
    "email": "person@example.com",
    "status": "pending",
    "confirmation_token": "JWT"
  },
  "status": "pending",
  "published_at": "datetime opcional",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

Os eventos suportados são `outbox_subscription_confirmation_requested` e `outbox_subscription_cancellation_requested`. O modelo de status define `pending`, `published` e `failed`; o processamento e as transições são trabalhos futuros.

## Ciclo de vida da inscrição

```text
novo e-mail ──solicitação──> pending ──confirma token──> subscribed
                                 ▲                            │
                                 │                            │ solicita cancelamento
                                 │                            ▼
                        renova solicitação <── unsubscribed <── confirma token
```

Os tokens de confirmação e cancelamento são JWTs com duração de 15 minutos. O token armazenado no documento da inscrição deve corresponder ao token recebido antes que a transição de estado seja aceita.

## API HTTP

Caminho base: `/api/v1`

| Método | Caminho | Corpo/query | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/subscription` | `{"email":"person@example.com"}` | `202 Accepted` |
| `POST` | `/subscription/confirm?token=...` | JWT como query parameter | `202 Accepted` |
| `POST` | `/unsubscription` | `{"email":"person@example.com"}` | `202 Accepted` |
| `POST` | `/unsubscription/confirm?token=...` | JWT como query parameter | `202 Accepted` |

```bash
curl -i -X POST http://localhost:8080/api/v1/subscription \
  -H "Content-Type: application/json" \
  -d '{"email":"person@example.com"}'
```

As respostas bem-sucedidas usam códigos de aplicação estáveis, como `SUBSCRIPTION_PENDING`, `SUBSCRIPTION_CONFIRMED`, `UNSUBSCRIPTION_PENDING` e `UNSUBSCRIPTION_CONFIRMED`.

Atualmente, o servidor aplica logging das requisições, recuperação de panics, IDs de requisição e um rate limit em memória de três requisições a cada 30 segundos. A configuração de CORS e o guard de tokens ainda não estão ativos. Há requisições prontas para execução em [`.http/subscriptions.http`](../.http/subscriptions.http).

## Configuração

| Variável | Obrigatória | Padrão | Finalidade |
| --- | --- | --- | --- |
| `SERVER_HOST` | não | `localhost` | Endereço de bind HTTP |
| `SERVER_PORT` | não | `8080` | Porta HTTP |
| `MONGO_URI` | sim | — | URI de conexão com um replica set MongoDB |
| `JWT_SECRET_KEY` | sim | — | Assina os JWTs de confirmação e cancelamento |
| `RESEND_SECRET_KEY` | atualmente sim | — | Reservada para o adapter planejado do Resend |

Não versione credenciais reais. Use valores locais em `.env` e mantenha apenas placeholders em `.env.example`.

## Comandos de desenvolvimento

```bash
make run    # go run ./cmd/api
make test   # go test ./...
make fmt    # go fmt ./...
make tidy   # go mod tidy
make lint   # golangci-lint run
make build  # compila bin/api
```

O Docker Compose compila a API e provisiona o MongoDB 8.0 como um replica set de nó único. O healthcheck do MongoDB inicializa `rs0`, e a API aguarda até que o nó se torne primário. Dentro da rede do Compose, `MONGO_URI` é sobrescrita automaticamente para usar o serviço `mongo`:

```bash
docker compose -f docker/docker-compose.yml up --build
```

A API é exposta na porta `8080`, o MongoDB na porta `27017`, e os dados são persistidos no volume nomeado `mongo_data`. Para encerrar os serviços preservando os dados, execute `docker compose -f docker/docker-compose.yml down`. Adicione `--volumes` somente quando quiser excluir intencionalmente o banco de dados local.

## Fluxo planejado com RabbitMQ e Resend

```text
outbox MongoDB → relay → exchange RabbitMQ → consumer de e-mail → API Resend
```

Cuidados de confiabilidade que devem ser preservados durante a implementação:

- Publicar com um ID de evento estável e usar publisher confirms.
- Tornar os consumers idempotentes; a entrega deve ser tratada como at least once.
- Reivindicar eventos atomicamente para que instâncias do relay não publiquem o mesmo trabalho simultaneamente.
- Incrementar `attempts`, repetir com backoff e enviar mensagens esgotadas para uma dead-letter queue.
- Marcar um evento como `published` somente após a confirmação do broker e preencher `published_at`.
- Manter o código específico do Resend atrás de uma porta outbound.
- Adicionar um índice para a consulta da outbox, por exemplo, sobre `status` e `created_at`.
- Propagar `event_id`, `aggregate_id` e request ID para observabilidade.

## Limitações atuais

- Ainda não existe um relay de polling/CDC para a outbox.
- RabbitMQ e Resend não estão conectados à aplicação.
- Atualmente não existem testes automatizados.
- O ambiente do Compose usa apenas um membro do replica set MongoDB e, portanto, não oferece alta disponibilidade para produção.
- Limpeza, retenção e reivindicação concorrente de eventos da outbox ainda não foram implementadas.
- A observabilidade está limitada aos logs HTTP e da aplicação.
