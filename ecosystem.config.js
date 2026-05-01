module.exports = {
  apps: [
    {
      name: "mirror-bot",
      script: "./run.sh",
      cwd: "/home/gabs/backups",

      // Roda como script simples (não fork infinito)
      autorestart: false,

      // Cron: meio-dia e meia-noite no horário local (UTC-3)
      // 00:00 BRT = 03:00 UTC  |  12:00 BRT = 15:00 UTC
      cron_restart: "0 3,15 * * *",

      // Mantém na lista mesmo sem estar rodando
      watch: false,

      // Logs separados por data
      out_file: "/home/gabs/backups/logs/mirror-bot-out.log",
      error_file: "/home/gabs/backups/logs/mirror-bot-err.log",
      merge_logs: false,
      log_date_format: "YYYY-MM-DD HH:mm:ss",

      // Não reinicia por falha (é uma tarefa cron, não um serviço)
      max_restarts: 0,
    },
  ],
};
