-- +goose Up
SELECT 'up SQL query';
ALTER TABLE arbitrage_spreads DROP COLUMN open_buy_price;
ALTER TABLE arbitrage_spreads DROP COLUMN open_sell_price;
ALTER TABLE arbitrage_spreads DROP COLUMN close_buy_price;
ALTER TABLE arbitrage_spreads DROP COLUMN close_sell_price;
ALTER TABLE arbitrage_spreads DROP COLUMN fees;
ALTER TABLE arbitrage_spreads DROP COLUMN profit;


-- +goose Down
SELECT 'down SQL query';
