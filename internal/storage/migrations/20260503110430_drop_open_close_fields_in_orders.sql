-- +goose Up
SELECT 'up SQL query';
ALTER TABLE orders DROP COLUMN open_buy_price;
ALTER TABLE orders DROP COLUMN open_sell_price;
ALTER TABLE orders DROP COLUMN close_buy_price;
ALTER TABLE orders DROP COLUMN close_sell_price;

-- +goose Down
SELECT 'down SQL query';
