-- +goose Up
SELECT 'up SQL query';
ALTER TABLE arbitrage_spreads ADD profit numeric(38, 18) NULL;

-- +goose Down
SELECT 'down SQL query';
