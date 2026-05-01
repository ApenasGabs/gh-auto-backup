# 🪞 Git Mirror Bot

## 📌 Propósito do Projeto

O **Git Mirror Bot** é um motor de sincronização contínua e automatizada, construído em **Go**, com foco absoluto em soberania de dados e redundância de código-fonte. Ele opera de forma autônoma para garantir que todo o portfólio de repositórios hospedado no GitHub possua um espelho exato, confiável e atualizado (1:1) no **GitLab** e em instâncias customizadas de **Gitea**.

Projetado para rodar silenciosamente em background via Docker, a ferramenta orquestra a comunicação entre as APIs das plataformas para assegurar que configurações críticas do ciclo de vida do código — especialmente a visibilidade e a privacidade dos repositórios — sejam rigorosamente mantidas entre a origem e os destinos.

## 🚀 Principais Features

*   **Sincronização 1:1 Absoluta:** Utiliza `git clone --mirror` e `git push --mirror` para replicar o estado exato da origem (todas as branches, tags, e histórico de commits intacto), tratando o GitHub como a fonte única da verdade.
*   **Gestão Dinâmica de Visibilidade:** Monitora ativamente o status de privacidade (`public`/`private`). Se um repositório for trancado ou aberto no GitHub, a alteração é propagada para os destinos via API.
*   **Segurança "SSH First":** Toda a sincronização de código é autenticada estritamente via chaves SSH, isolando credenciais do fluxo de clone/push.
*   **Provisionamento Automático (Auto-Discovery):** Descobre automaticamente novos repositórios e os cria nos destinos (GitLab/Gitea) sem intervenção manual.
*   **Resiliência e Self-Healing:** Implementação de `context` para `Graceful Shutdown` e tratamento de erros robusto.

## 🛠️ Stack Tecnológico
*   **Linguagem:** Go (Golang).
*   **Infraestrutura:** Docker & Docker Compose.
*   **Integrações:** APIs oficiais para GitHub, GitLab e Gitea SDK.

## 🏁 Como Rodar

### 1. Pré-requisitos
*   **Docker & Docker Compose** (opcional, para execução em container).
*   **Node.js & PM2** (opcional, para execução via daemon/cron no host).
*   **Go 1.21+** (para compilação manual).
*   **Chave SSH configurada:** A chave SSH da sua máquina host (geralmente em `~/.ssh/id_rsa`) deve estar adicionada como "SSH Key" no GitHub, GitLab e Gitea para permitir o clone/push sem senha.

### 2. Configuração
Crie um arquivo `.env` baseado no exemplo:
```bash
cp .env.example .env
```
Edite o `.env` com suas credenciais:
*   `GITHUB_TOKEN`: Token com permissão de leitura de repositórios.
*   `GITLAB_TOKEN`: Token com permissão de criação de projetos.
*   `GITEA_TOKEN`: Token da sua instância Gitea.
*   `GITEA_URL` & `GITEA_SSH_HOST`: Endereços da sua instância self-hosted.

### 3. Compilação
Antes de rodar em qualquer ambiente, compile o binário:
```bash
go build -o mirror-bot
```

### 4. Modos de Execução (Testes)
*   **Dry Run (Comparar e listar):**
    ```bash
    ./mirror-bot --list
    ```
*   **Modo Interativo (Seleção manual):**
    ```bash
    ./mirror-bot --interactive
    ```
*   **Modo Full Auto:**
    ```bash
    ./mirror-bot --all
    ```

### 5. Produção: Opção A (Docker)
Recomendado para ambientes isolados.
```bash
# Executar a sincronização de todos os repositórios (--all) via Docker
docker compose run --rm mirror-bot ./mirror-bot --all
```
*(Dica: Você pode adicionar este comando em um `cronjob` no host).*

### 6. Produção: Opção B (PM2) - Recomendado
Ideal para servidores Linux (VPS). Utiliza o `ecosystem.config.js` para gerenciar execuções agendadas via Cron.

```bash
# Iniciar o processo via PM2
pm2 start ecosystem.config.js

# Salvar a lista para persistir após reboots do sistema
pm2 save

# Monitorar a execução e logs
pm2 logs mirror-bot
```

**Comandos Úteis PM2:**
*   `pm2 status`: Lista os processos e status do agendamento.
*   `pm2 stop mirror-bot`: Interrompe o agendamento.
*   `pm2 trigger mirror-bot`: (Se configurado) ou apenas `pm2 restart mirror-bot` para forçar uma execução agora.

## ⚙️ Variáveis de Ambiente Adicionais
*   `SYNC_INTERVAL`: Intervalo entre as sincronizações (ex: `1h`, `30m`). Default: `1h`.
*   `WORKER_COUNT`: Número de repositórios sincronizados em paralelo. Default: `5`.
*   `STORAGE_DIR`: Diretório interno onde os mirrors locais são armazenados. Default: `./storage`.