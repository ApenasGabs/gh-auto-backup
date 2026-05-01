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

### 3. Execução (Local / Testes)
Antes de deixar o bot rodando no piloto automático, é recomendado rodar as etapas iniciais localmente (requer Go instalado):

```bash
# Compilar o binário
go build -o mirror-bot
```

**Modos de Execução Disponíveis:**
*   **Dry Run (Comparar e listar):** Lista todos os repositórios do GitHub e verifica quais faltam no GitLab, gerando uma tabela comparativa sem alterar nada.
    ```bash
    ./mirror-bot --list
    ```
*   **Modo Interativo (Seleção manual):** Exibe um checklist para você selecionar quais repositórios específicos deseja migrar (ótimo para "Amostras Grátis" e testes iniciais).
    ```bash
    ./mirror-bot --interactive
    ```
*   **Modo Full Auto:** Inicia o processo automático de sincronização para todos os repositórios listados.
    ```bash
    ./mirror-bot --all
    ```

### 4. Execução em Produção (Docker)
Após testar localmente, você pode empacotar a execução no Docker. 
Como a ferramenta agora requer uma flag de execução, você pode inicializar o container rodando apenas o modo desejado:

```bash
# Executar a sincronização de todos os repositórios (--all) via Docker
docker compose run --rm mirror-bot ./mirror-bot --all
```
*(Dica: Como o bot realiza a sincronização e finaliza sua execução, você pode adicionar este comando em um `cronjob` para rodar de hora em hora em seu servidor).*

## ⚙️ Variáveis de Ambiente Adicionais
*   `SYNC_INTERVAL`: Intervalo entre as sincronizações (ex: `1h`, `30m`). Default: `1h`.
*   `WORKER_COUNT`: Número de repositórios sincronizados em paralelo. Default: `5`.
*   `STORAGE_DIR`: Diretório interno onde os mirrors locais são armazenados. Default: `./storage`.