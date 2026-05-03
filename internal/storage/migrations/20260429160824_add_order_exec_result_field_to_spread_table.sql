-- +goose Up
SELECT 'up SQL query';
ALTER TABLE arbitrage_spreads ADD open_buy_price numeric(38, 18) NULL;
ALTER TABLE arbitrage_spreads ADD open_sell_price numeric(38, 18) NULL;
ALTER TABLE arbitrage_spreads ADD close_buy_price numeric(38, 18) NULL;
ALTER TABLE arbitrage_spreads ADD close_sell_price numeric(38, 18) NULL;
ALTER TABLE arbitrage_spreads ADD fees numeric(38, 18) NULL;

-- +goose Down
SELECT 'down SQL query';
