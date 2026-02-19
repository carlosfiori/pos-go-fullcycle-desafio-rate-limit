# Rate Limiter em Go com Redis

Implementação de rate limiter HTTP em Go usando Redis como backend. Suporta limitação por IP e por token de acesso.


## Como Funciona

O rate limiter usa o algoritmo **fixed-window**:

1. Cada requisição incrementa um contador no segundo atual
2. Se exceder o limite, o identificador (IP ou token) é bloqueado
3. IPs/tokens bloqueados são rejeitados imediatamente
4. O bloqueio expira após o tempo configurado

## Configuração

Crie um arquivo `.env` na raiz do projeto:

```bash
# Redis
REDIS_ADDR=redis:6379
REDIS_PASSWORD=

# Limite por IP
RATE_LIMIT_IP=10                        # Requisições por segundo
RATE_LIMIT_IP_BLOCK_DURATION=300        # Tempo de bloqueio em segundos

# Tokens (formato: token:limite:bloqueio)
RATE_LIMIT_TOKENS=abc123:100:300,xyz789:50:600
```

## Como Rodar

### Com Docker Compose

```bash
docker-compose up --build
```

A aplicação estará disponível em `http://localhost:8080`.



## Testando

### Testes Automatizados

```bash
go test ./...
```

### Testes Manuais

**Teste básico:**
```bash
curl http://localhost:8080/
```

**Testar limite (script simples):**
```bash
./test_rate_limit.sh 15
```

**Com token:**
```bash
./test_rate_limit.sh 20 abc123
```

### Comportamento Esperado

- Dentro do limite: `200 OK`
- Excedeu o limite: `429 Too Many Requests`
- Após expirar bloqueio: Volta ao normal

