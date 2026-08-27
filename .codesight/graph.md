# Dependency Graph

## Most Imported Files (change these carefully)

- `JoblessYu/internal/job` — imported by **7** files
- `JoblessYu/internal/config` — imported by **6** files
- `log/slog` — imported by **6** files
- `encoding/json` — imported by **5** files
- `JoblessYu/internal/bot` — imported by **4** files
- `net/http` — imported by **3** files
- `JoblessYu/internal/scraper` — imported by **2** files
- `encoding/hex` — imported by **2** files
- `path/filepath` — imported by **2** files
- `os/signal` — imported by **1** files
- `crypto/ed25519` — imported by **1** files
- `net/url` — imported by **1** files
- `sync/atomic` — imported by **1** files
- `crypto/md5` — imported by **1** files
- `database/sql` — imported by **1** files
- `os/exec` — imported by **1** files

## Import Map (who imports what)

- `JoblessYu/internal/job` ← `cmd\bot\main.go`, `cmd\lambda-bot\main.go`, `cmd\lambda-pipeline\main.go`, `internal\bot\bot.go`, `internal\bot\embeds.go` +2 more
- `JoblessYu/internal/config` ← `cmd\bot\main.go`, `cmd\lambda-bot\main.go`, `cmd\lambda-pipeline\main.go`, `cmd\migrate\main.go`, `internal\bot\bot.go` +1 more
- `log/slog` ← `internal\bot\notifier.go`, `internal\job\colly_scraper.go`, `internal\job\enricher.go`, `internal\job\groq.go`, `internal\job\store.go` +1 more
- `encoding/json` ← `cmd\lambda-bot\main.go`, `cmd\lambda-pipeline\main.go`, `internal\bot\notifier.go`, `internal\job\groq.go`, `internal\job\store.go`
- `JoblessYu/internal/bot` ← `cmd\bot\main.go`, `cmd\lambda-bot\main.go`, `cmd\lambda-pipeline\main.go`, `internal\scraper\scraper.go`
- `net/http` ← `cmd\bot\main.go`, `cmd\lambda-bot\main.go`, `internal\bot\notifier.go`
- `JoblessYu/internal/scraper` ← `cmd\bot\main.go`, `cmd\lambda-pipeline\main.go`
- `encoding/hex` ← `cmd\lambda-bot\main.go`, `internal\job\store.go`
- `path/filepath` ← `cmd\migrate\main.go`, `internal\scraper\scraper.go`
- `os/signal` ← `cmd\bot\main.go`
