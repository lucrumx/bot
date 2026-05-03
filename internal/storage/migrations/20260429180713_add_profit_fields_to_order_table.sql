-- +goose Up
SELECT 'up SQL query';
ALTER TABLE orders ADD open_buy_price numeric(38, 18) NULL;
ALTER TABLE orders ADD open_sell_price numeric(38, 18) NULL;
ALTER TABLE orders ADD close_buy_price numeric(38, 18) NULL;
ALTER TABLE orders ADD close_sell_price numeric(38, 18) NULL;
ALTER TABLE orders ADD fees numeric(38, 18) NULL;

-- +goose Down
SELECT 'down SQL query';
