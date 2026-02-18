# LucrumX - Multi-Exchange Crypto Trading Engine

![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)
![Build Status](https://img.shields.io/github/actions/workflow/status/lucrumx/bot/tests.yml?branch=main)

A high-performance real-time trading engine for crypto market analysis and automated detection. The system monitors multiple exchanges simultaneously to identify market anomalies and arbitrage opportunities.

## 🤖 System Components

### 1. Pump Detector
Real-time detection of significant price impulses on futures markets.
- **Algorithm**: Continuous sliding window on a ring-buffer with a Gap Filling mechanism.
- **Adaptive Thresholds**: Uses market-dynamic factors to filter noise.
- **Intelligent Alerting**: Multi-level notifications (Alert Step) and signal cooldowns.

### 2. Arbitrage Bot
Real-time spread monitoring between different exchanges.
- **Symbol Discovery**: Automatic cross-exchange instrument intersection.
- **Normalization**: Standardizes disparate symbol formats (e.g., `BTC-USDT` vs `BTCUSDT`).
- **Spread Detection**: Calculates clean (Net) spreads with stale price protection (MaxAge) and signal throttling.

## 📊 API & Web Interface

The system includes a centralized API and an embedded web interface for monitoring and management.
- **Backend API (`cmd/api`)**: Provides REST endpoints for data retrieval and future control. All API requests are prefixed with `/api/`.
- **Embedded Frontend**: A Single Page Application (SPA) built with **Nuxt.js v4 (Vue 3)**, compiled to static assets, and embedded directly into the Go binary using `go:embed`.
- **Routing**: Requests to `/api/*` are handled by the Go backend. All other routes are served by the embedded Nuxt.js SPA, allowing client-side routing.

## 🚀 Core Features

- **Multi-Exchange Support**: Native integrations for **Bybit (V5)** and **BingX (V2 Swap)** via optimized WebSocket clients.
- **WebSocket Orchestration**: Centralized `WSManager` handling chunked subscriptions, non-blocking backpressure management, and connection lifecycle.
- **O(1) Detection Performance**: Logic is optimized for constant time complexity, regardless of the number of monitored tickers.
- **Hybrid Math Engine**: Employs `float64` for high-throughput stream processing and `shopspring/decimal` for precision-critical filtering and financial logic.

## 🛠 Tech Stack

- **Language**: Go 1.24.
- **Frontend**: Nuxt.js v4 (Vue 3) + Tailwind CSS + DaisyUI.
- **Transport**: REST & WebSocket API (Bybit, BingX).
- **Notifications**: Telegram Bot API.
- **Logging**: `zerolog` (structured JSON logging).
- **Storage**: PostgreSQL + GORM (Management API).

## 📈 Performance (Observed)

The engine is built for dense data streams with low latency. Values below are current observed throughput from runtime logs, not peak limits:
- **Pump Bot (Bybit)**: ~1400+ trades/sec on MacBook Pro M2 Pro.
- **Arbitrage Bot (Bybit + BingX)**: Monitoring **439 common pairs**. Current aggregate throughput is about **2600-2800 events/sec** (for example, 2628 and 2836 events/sec in consecutive 60s windows).

## 🏗 Project Structure

- `cmd/api/`: Entry point for the combined API and Web Interface.
- `cmd/pumpbot/`: Pump Detector entry point.
- `cmd/arbitragebot/`: Arbitrage Bot entry point.
- `internal/exchange/`: 
    - `client/`: Bybit and BingX exchange adapters.
    - `pumpbot/`: Core logic for impulse detection.
    - `arbitragebot/`: Core logic for spread monitoring and API handlers.
    - `ws_manager.go`: Unified WebSocket connection manager.
- `internal/ui/`: Embedded Nuxt.js frontend assets and serving logic.
- `internal/notifier/`: Telegram notification system.
- `internal/config/`: Configuration management (YAML + ENV).

## 🧪 Testing

- **Unit Tests**: Full coverage for Sliding Window, Spread Detection, and Cooldown logic.
- **Integration Tests**: Docker-based database tests and Telegram mock servers.
- **Strict Linting**: Enforced via `golangci-lint`.

## ⚙️ Configuration

The system uses a flexible configuration approach with the following priority: 
1. **Command Line Flag**: `--config path/to/config.yaml`
2. **Default File**: `config.yaml` in the root directory.
3. **Environment Variables**: Loaded from `.env` or system environment if no YAML file is found.

## 🚦 Getting Started

1. **Clone & Setup**:
   ```bash
   git clone https://github.com/lucrumx/bot.git
   cp .env.dist .env
   ```
2. **Install Go Dependencies**: `go mod download`
3. **Build and Embed Frontend**:
   ```bash
   cd internal/ui/frontend
   npm install
   npm run generate
   cd ../../.. # Go back to project root
   ```
4. **Run API & Web Interface**: `go run cmd/api/main.go`
5. **Run Arbitrage Bot**: `go run cmd/arbitragebot/main.go`
6. **Run Pump Bot**: `go run cmd/pumpbot/main.go`
