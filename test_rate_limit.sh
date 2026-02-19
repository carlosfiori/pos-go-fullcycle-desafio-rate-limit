#!/bin/bash

# Script simples para testar o Rate Limiter
# Uso: ./test_rate_limit.sh [quantidade] [token]
# Exemplo: ./test_rate_limit.sh 15
# Exemplo com token: ./test_rate_limit.sh 20 abc123

QUANTIDADE=${1:-15}
TOKEN=${2:-}
URL="http://localhost:8080/"

echo "Enviando $QUANTIDADE requisições para $URL"
if [ -n "$TOKEN" ]; then
    echo "Usando token: $TOKEN"
fi
echo ""

for i in $(seq 1 $QUANTIDADE); do
    if [ -n "$TOKEN" ]; then
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "API_KEY: $TOKEN" $URL)
    else
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" $URL)
    fi
    echo "Request $i: $STATUS"
done
