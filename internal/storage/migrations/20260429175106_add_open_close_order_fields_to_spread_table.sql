-- +goose Up
SELECT 'up SQL query';
ALTER TABLE arbitrage_spreads ADD open_buy_order_id uuid NULL;
ALTER TABLE arbitrage_spreads ADD open_sell_order_id uuid NULL;
ALTER TABLE arbitrage_spreads ADD close_buy_order_id uuid NULL;
ALTER TABLE arbitrage_spreads ADD close_sell_order_id uuid NULL;

-- +goose Down
SELECT 'down SQL query';
