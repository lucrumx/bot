# Pump Detector (Linear Futures) на примере Bybit

Для других бирж стратегия (см раздел ниже) аналогична, другие пороговые значения, которые подбираются
экспериментально, см детально "адаптивный thresholds" ниже

## Цель

Получать real-time сделки (matches) по малоликвидным фьючерсным инструментам Bybit и выявлять аномальные всплески активности, характерные для начальной фазы пампов.

Допущения:
- потеря части данных допустима
- при рестарте система начинает мониторинг заново
- приоритет: низкая задержка и простота

---

## Источник данных

### Bybit API
- REST: `/v5/market/tickers?category=linear`
- WebSocket: `publicTrade.{symbol}`

Используется только публичный поток сделок.

---

## Архитектура (In-Memory Pipeline)

```text
+------------------------+
|   Bybit WebSocket      |
|  (publicTrade.{symbol})|
+-----------+------------+
            |
            v
+------------------------+
|   WS Ingest Worker     |
| - shard symbols        |
| - normalize trades     |
| - burst control        |
|   (buffered channel)   |
+-----------+------------+
            |
            v
   buffered channel ch
  (например, size=10000, non-blocking send)
            |
            v
+------------------------+
|  Pump Detector         |
| - Sliding windows 1s/3s/5s|
| - Calculate metrics:    |
|   volume, trade count,  |
|   price change          |
| - Compare с thresholds  |
|   (adaptive: Vmin/Tmin)|
+-----------+------------+
            |
            v
+------------------------+
|   Signal Handler       |
| - generate alert       |
| - log, notify UI, etc. |
+------------------------+
```

### Компоненты

#### Symbol selector 
    
__На старте и периодически:__
- запрос /v5/market/tickers
- фильтрация инструментов по ликвидности

__Пример фильтров:__
- turnover24h < 5–20 млн USDT
- openInterest < 1–3 млн USDT
- валидная цена

__Результат:__
- 300–500 малоликвидных символов

#### WebSocket ingest
__Коннекты__
- до 200 topics на один WS
- 3–4 WS соединения
  - 120–150 символов на коннект

__Endpoint:__ `wss://stream.bybit.com/v5/public/linear`

__Каждый ingest:__
 - подключение к WS
 - подписка на topics (топик publicTrade)
 - обработка и нормализация данных
 - отправка данных в Go Channel (Buffered)

---

### Нормализация trade

Цели:
  - убрать лишние поля
  - привести данные к компактному формату
  - упростить downstream обработку 

__Raw trade от Bybit__
```aiignore
{
  "T": 1700000000000,
  "s": "BTCUSDT",
  "p": "43000.5",
  "v": "0.12",
  "S": "Buy",
  "i": "123456",
  "m": false
}
```
__Normalized trade__  
Go структура:  
для цен использовать `github.com/shopspring/decimal`
```go
type Trade struct {
        Symbol string
        Ts     int64 // Unix timestamp
        Price  decimal.Decimal
        Size   decimal.Decimal
        Side   uint8 // 0 - buy, 1 - sell
        Usdt   decimal.Decimal // price * size
}
```
Примечания:  
- price * size считается сразу
- side кодируется как 0/1

---

### Burst control  
Проблема:
- bursts тиков

Без него:  
- buffered channel или очередь быстро переполняется
- растёт latency, возможны зависания или падения бота

Нужен механизм защиты пайплайна от резких всплесков тиков (trades), которые Bybit присылает пачками.

Решение:
1.      Buffered channel между WS и Detector
2.      Non-blocking send
3.      Drop policy при overflow

Пример (non-blocking send в Go для канала (channel)):
```go
select {
case ch <- trade:   // пытаемся отправить trade в канал ch
default:            // если канал занят (буфер полон или никто не читает)
// drop trade   // просто пропускаем, не блокируемся
}
```

---

### Детектор пампа

__Sliding windows__  
Для каждого символа:
- 1s
- 3s
- 5s

Считается:
- trade count
- volume (USDT)
- price delta

---

### Thresholds (пороги)

___Пороги на примере ByBit, для разных бирж они разные, так как разные объемы торгуются.___

#### Абсолютные пороги

Для инструментов с turnover24h ~3–10 млн USDT:

__Trade count__
- норма: 1–5 trades/sec
- сигнал: ≥ 20–30 trades/sec

__Volume__
- ≥ 30k–80k USDT за 1s
- ≥ 100k–300k USDT за 3s

__Price velocity__
- ≥ 0.3–0.6% за 1s
- ≥ 1–1.5% за 2–3s

---

### Комбинированный cигнал

```sql 
volume_1s > Vmin
AND trades_1s > Tmin
AND price_move_1s > Pmin
```

#### Адаптивные thresholds
```text
         Market Data (trades)
                |
                v
  -------------------------------
  |  Calculate metrics per window |
  |  - Volume (USDT)              |
  |  - Trade count                |
  |  - Price change / velocity    |
  -------------------------------
                |
                v
  --------------------------------------
  |  Compute adaptive thresholds        |
  |  - Avg volume over 5 min            |
  |  - Avg trade count over 5 min       |
  |  - Scale with factor k (8–12)       |
  |                                     |
  |  Vmin = max(abs_min, avg_volume* k) |
  |  Tmin = max(abs_min, avg_trades* k) |
  --------------------------------------
                |
                v
  ----------------------------------
  |  Compare metrics with thresholds |
  |  - Volume > Vmin                 |
  |  - Trades > Tmin                 |
  |  - Price move > Pmin             |
  -----------------------------------
                |
          True / False
           /      \
          v        v
   ----------------   ----------------
   |  Signal ON  |   |  No signal    |
   ----------------   ----------------
```

k = 8, 12, abs_min = 15k–20k USDT

Принцип работы:
1.      Берём статистику за прошлые N минут (обычно 5 минут).
2.      Считаем средний объём и среднее количество сделок.
3.      Умножаем на коэффициент k = 8–12 для «порогового уровня».
4.      Берём максимум между абсолютным минимумом (abs_min, например 15–20k USDT) и рассчитанным адаптивным значением.
5.      Используем этот адаптивный порог для генерации сигналов.

k — это коэффициент масштабирования для адаптивных порогов.

Для малоликвидной линейной секции Bybit:
- k ≈ 8–12 - отделяет фоновые колебания от реальных всплесков.
- Выбор точного числа зависит от:
  - средней ликвидности инструмента
  - частоты сделок
  - времени окна (1s, 3s, 5s)

__k__ — регулятор чувствительности детектора, подбирается экспериментально.

---

### Фильтры
__Минимум 2 из 3 условий:__  
- volume
- trades
- price velocity

__Price follow-through:__
- цена удерживается ≥ 0.5–1s 

__Direction consistency:__
- ≥ 60–70% trades в одну сторону

__Исключение single-trade spikes:__
- trades_1s ≥ Tmin обязательно