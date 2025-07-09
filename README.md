# 🐔 aGOrinha 2025

Implementação do desafio da **Rinha de Backend 2025** utilizando **Go (v1.24)**, com foco em desempenho, concorrência eficiente e fallback resiliente para processadores de pagamento.

> Repositório oficial da Rinha de Backend: [zanfranceschi/rinha-de-backend-2025](https://github.com/zanfranceschi/rinha-de-backend-2025)

## 🔥 Descrição

Esta solução implementa uma API HTTP com dois endpoints principais:

- `POST /payments`: recebe requisições de pagamento e as encaminha para o Payment Processor mais adequado (`default` ou `fallback`), priorizando menor taxa e maior disponibilidade.
- `GET /payments-summary`: retorna o resumo dos pagamentos processados entre dois períodos (default vs fallback).

Pagamentos são processados com lógica de fallback automática para o segundo processador em caso de falha ou timeout. O sistema utiliza:

- Pool de workers para paralelismo.
- Persistência em marquivo (com flock).

## 📁 Estrutura

```bash
rinha2025/
├── cmd/
│   └── server/             # Entrypoint do servidor HTTP
├── internal/
│   ├── api/                # Handlers da API
│   ├── client/             # Cliente para processadores de pagamento
│   ├── core/               # Lógica de negócios (Payment)
│   ├── store/              # Implementações de persistência (arquivo)
│   └── worker/             # Pool de workers
├── go.mod
└── go.sum
````

## ⚙️ Tecnologias Utilizadas

* Linguagem: **Go 1.24.4**
* Web server: **fasthttp**
* Persistência: **Em memória** (arquivo `.json`)
* Load balancer: **NGINX**
* Comunicação com Processadores: HTTP via `fasthttp.Client`
* Orquestração: **Docker Compose**

## 🧠 Estratégia de Processamento

* **Assíncrono com fallback inteligente**: se `SendToDefault` falhar (exceto erro 422), tenta `SendToFallback`.
* **Registro de sucesso**: apenas pagamentos com status 2XX são registrados na store.
* **Evita inconsistência**: pagamentos com erro 422 não são registrados em nenhuma store.

## 🧪 Endpoints

### POST /payments

```json
{
  "correlationId": "uuid-1234",
  "amount": 19.90
}
```

### GET /payments-summary

Query params:

* `from` (opcional)
* `to` (opcional)

Resposta:

```json
{
  "default": {
    "totalRequests": 10,
    "totalAmount": 199.0
  },
  "fallback": {
    "totalRequests": 2,
    "totalAmount": 39.8
  }
}
```
