# LucrumX - High-Performance Crypto Pump Detector

![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)
![Build Status](https://img.shields.io/github/actions/workflow/status/lucrumx/bot/tests.yml?branch=main)

A high-frequency trading bot designed for real-time market anomaly detection on **Bybit Linear Futures** (USDT). The system monitors price and volume dynamics to identify explosive movements (pumps) using adaptive threshold logic.

## 🚀 Core Features

- **Real-time Detection Engine**: 
  - **Flash Pump (1s)**: Captures immediate price/volume spikes.
  - **Momentum Pump (3s)**: Detects sustained trend-based movements.
- **Adaptive Thresholds**: Dynamic signal filtering based on a 5-minute rolling average of volume and trade count, multiplied by a configurable `K-Factor`.
- **High-Performance Sliding Window**: Custom ring-buffer implementation for O(1) statistics calculation.
- **WebSocket Orchestration**: Efficient management of multiple ticker streams with automatic reconnection.
- **Modular Architecture**: Strict separation of concerns using Vertical Slicing and Dependency Injection.

## 🛠 Tech Stack

- **Language**: Go 1.24 (utilizing latest concurrency primitives).
- **Transport**: Bybit V5 REST & WebSocket API.
- **Web Framework**: Gin Gonic (for Management API).
- **Storage**: PostgreSQL + GORM.
- **Math**: `shopspring/decimal` for fixed-point financial precision.
- **Logging**: `zerolog` (structured JSON logging).

## 🏗 Project Structure

The project follows the standard Go layout with a focus on domain-driven design:

- `cmd/`: Application entry points (`bot` and `api`).
- `internal/exchange/`: Core trading domain, exchange adapters (Bybit), and engine.
- `internal/auth/ & internal/users/`: Authentication and user management services.
- `internal/models/`: GORM database entities.
- `internal/storage/`: Database initialization and migrations.

## 🧪 Development & Quality Control

### Testing Strategy
- **Unit Tests**: Coverage for core logic (Sliding Window, Pump Detection) using `testify` and `mockery`.
- **Integration Tests**: End-to-end database and API testing using **Docker Compose** for ephemeral environments.

### Tooling (Makefile)
The project includes a robust `Makefile` for a standardized development workflow:
- `make lint`: Runs `golangci-lint` with strict configurations.
- `make test`: Executes all unit tests.
- `make test-integration`: Runs integration tests in a Docker environment.
- `make run-bot`: Starts the pump detector engine.
- `make build`: Compiles production binaries.

## ⚙️ Configuration

System behavior is managed via environment variables:
- `K_FACTOR`: Multiplier for adaptive volume thresholds.
- `ABS_MIN_VOLUME`: Absolute minimum USDT volume to filter noise.
- `DB_DSN`: PostgreSQL connection string.
- `JWT_SECRET`: Secret key for API authentication.

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
