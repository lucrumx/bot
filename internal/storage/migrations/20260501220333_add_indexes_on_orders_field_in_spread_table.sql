-- +goose Up
SELECT 'up SQL query';
CREATE INDEX "idx_spread_open_buy_order_id" on "arbitrage_spreads" ("open_buy_order_id" ASC);
CREATE INDEX "idx_spread_open_sell_order_id" on "arbitrage_spreads" ("open_sell_order_id" ASC);
CREATE INDEX "idx_spread_close_buy_order_id" on "arbitrage_spreads" ("close_buy_order_id" ASC);
CREATE INDEX "idx_spread_close_sell_order_id" on "arbitrage_spreads" ("close_sell_order_id" ASC);

-- +goose Down
SELECT 'down SQL query';
