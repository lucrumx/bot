package arbitragebot

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
)

//mockery:generate: true
type BalanceStore interface {
	GetForAsset(exchange string, asset string) (models.Balance, bool)
	Get(exchange string) ([]models.Balance, bool)
	Set(exchange string, balances []models.Balance)
}

type balanceStore struct {
	mu       sync.RWMutex
	balances map[string]map[string]models.Balance // exchange -> currency -> balance
	logger   zerolog.Logger
}

func newBalanceStore(logger zerolog.Logger) *balanceStore {
	return &balanceStore{
		logger:   logger,
		balances: make(map[string]map[string]models.Balance),
	}
}

func (bs *balanceStore) Start(ctx context.Context, clients []exchange.Provider) {
	bs.retrieveBalances(ctx, clients)

	go func() {
		ticker := time.NewTicker(time.Second * 20)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				bs.retrieveBalances(ctx, clients)
			}
		}
	}()
}

func (bs *balanceStore) retrieveBalances(ctx context.Context, clients []exchange.Provider) {
	for _, clients := range clients {
		balances, err := clients.GetBalances(ctx)

		if err != nil {
			bs.logger.Warn().Err(err).Msgf("can`t get balance for exchange %s", clients.GetExchangeName())
			continue
		}

		bs.Set(balances)
	}
}

func (bs *balanceStore) GetForAsset(exchangeName string, asset string) (models.Balance, bool) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	assets, ok := bs.balances[exchangeName]
	if !ok {
		return models.Balance{}, false
	}

	balance, ok := assets[asset]
	if !ok {
		return models.Balance{}, false
	}

	return balance, true
}

func (bs *balanceStore) Get(exchangeName string) ([]models.Balance, bool) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	assets, ok := bs.balances[exchangeName]
	if !ok {
		return nil, false
	}

	b := make([]models.Balance, 0, len(assets))

	for _, balance := range assets {
		b = append(b, balance)
	}

	return b, true
}

func (bs *balanceStore) Set(balances []models.Balance) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	for _, balance := range balances {
		exchangeName := balance.ExchangeName

		if bs.balances[exchangeName] == nil {
			bs.balances[exchangeName] = make(map[string]models.Balance)
		}

		bs.balances[exchangeName][balance.Asset] = balance
	}
}
