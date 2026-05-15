-- +goose Up
SELECT 'up SQL query';
ALTER TABLE arbitrage_spreads ADD closed_at timestamptz NULL;

-- +goose Down
SELECT 'down SQL query';
