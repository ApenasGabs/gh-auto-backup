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
*   **Docker & Docker Compose** instalados.
*   **Chave SSH** configurada: A chave SSH da sua máquina host (geralmente em `~/.ssh/id_rsa`) deve estar adicionada como "Deploy Key" ou "SSH Key" no seu perfil do GitHub, GitLab e Gitea para permitir o clone/push sem senha.

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

### 3. Execução
Para subir o serviço em background:
```bash
docker compose up -d --build
```

Para acompanhar os logs de sincronização:
```bash
docker compose logs -f
```

## ⚙️ Variáveis de Ambiente Adicionais
*   `SYNC_INTERVAL`: Intervalo entre as sincronizações (ex: `1h`, `30m`). Default: `1h`.
*   `WORKER_COUNT`: Número de repositórios sincronizados em paralelo. Default: `5`.
*   `STORAGE_DIR`: Diretório interno onde os mirrors locais são armazenados. Default: `./storage`.