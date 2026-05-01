#!/usr/bin/env bash
set -euo pipefail

# Garante que roda a partir do diretório do script
cd "$(dirname "$0")"

# Carrega variáveis do .env (ignora linhas de comentário e linhas vazias)
if [ -f .env ]; then
  set -a
  # shellcheck source=/dev/null
  source .env
  set +a
fi

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Iniciando mirror-bot --all"
./mirror-bot --all
echo "[$(date '+%Y-%m-%d %H:%M:%S')] mirror-bot finalizado com código $?"
