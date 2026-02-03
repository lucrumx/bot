# LucrumX - High-Performance Crypto Pump Detector

![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)
![Build Status](https://img.shields.io/github/actions/workflow/status/lucrumx/bot/tests.yml?branch=main)

A high-performance real-time market anomaly detector for **Bybit Linear Futures** (USDT). The system monitors price dynamics to identify significant momentum movements (pumps) using optimized sliding window logic and intelligent alerting.

## 🚀 Core Features

- **Momentum Detection (30m)**: Tracks significant price impulses over configurable intervals (e.g., 15% growth over 15 minutes).
- **Intelligent Alerting System**: 
  - **Alert Step**: Sends follow-up notifications if the price continues to rise (e.g., every +5% after the initial signal).
  - **Cooldown**: Prevents spam and redundant signals for the same price movement.
- **Continuous Sliding Window**: Custom ring-buffer implementation with **Gap Filling** mechanism (automatically populates missing data points during low liquidity), ensuring seamless analysis.
- **Multi-channel Notifier**: Telegram integration for instant alerts with HTML formatting support.
- **O(1) Performance**: Detection logic is optimized for constant time complexity, allowing monitoring of hundreds of tickers with minimal CPU overhead.
- **Parallel Trade Processing**: Worker pool with deterministic symbol hashing; per-worker windows replace a global mutex and improve throughput under load.
- **WebSocket Orchestration**: Efficient data stream management with automatic reconnection and chunked subscription handling.

## 🛠 Tech Stack

- **Language**: Go 1.24.
- **Transport**: Bybit V5 REST & WebSocket API.
- **Notifications**: Telegram Bot API.
- **Math**: `shopspring/decimal` for fixed-point financial precision.
- **Logging**: `zerolog` (structured JSON logging).
- **Storage**: PostgreSQL + GORM (for Management API).

## 📈 Performance (Observed)

Real‑time processing of hundreds of trades per second with low latency.  
On MacBook Pro M2 Pro, the engine processes ~1400 trades/sec (processed) with a 60s reporting window, Bybit Linear Futures, measured over ~24h. Processing is parallelized via a hash‑partitioned worker pool.

## 🏗 Project Structure

The project follows a modular Go layout with a focus on domain-driven design:

- `cmd/`: Application entry points (`bot` and `api`).
- `internal/exchange/`: Core trading domain, exchange adapters (Bybit), and detection engine.
- `internal/notifier/`: Notification system (Notifier interface and Telegram implementation).
- `internal/utils/`: Common utilities, environment, and configuration helpers.
- `internal/auth/ & internal/users/`: Authentication and user management services.

## 🧪 Testing & Quality Control

- **Unit Tests**: Comprehensive coverage for core logic (Sliding Window, Gap Filling, Alert Logic).
- **Integration Tests**: Telegram API integration via mock servers and database testing in Docker environments.
- **Strict Linting**: Enforced code quality via `golangci-lint` with strict configurations.

## ⚙️ Configuration (.env)

System behavior is managed via environment variables:

- `PUMP_INTERVAL`: Analysis interval in seconds (e.g., 900 for 15 minutes).
- `TARGET_PRICE_CHANGE`: Target percentage growth for the initial signal (e.g., 15).
- `ALERT_STEP`: Percentage step for follow-up notifications (e.g., 5).
- `CHECK_INTERVAL`: Frequency of price checks in seconds.
- `FILTER_TICKERS_TURNOVER`: Filter tickers by 24h turnover (USDT).
- `TELEGRAM_BOT_TOKEN`: Your bot token from @BotFather.
- `TELEGRAM_CHAT_ID`: Destination chat or channel ID.

## 🚦 Getting Started

1. **Clone & Setup**:
   ```bash
   git clone https://github.com/lucrumx/bot.git
   cp .env.dist .env
   ```
2. **Install Dependencies**:
   ```bash
   go mod download
   ```
3. **Run Tests**:
   ```bash
   make test
   ```
4. **Launch Bot**:
   ```bash
   make run-bot
   ```
