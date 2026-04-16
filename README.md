# Seller Trust Map

Дипломный проект: сервис для анализа репутации продавцов и проверки достоверности рейтинга карточек товара на маркетплейсах.

## Стек

- Go backend API
- Python FastAPI anti-fraud service
- PostgreSQL
- Redis
- браузерное расширение
- Docker Compose

## Структура проекта

- `backend-go` — основной API, обработка URL, расчет trust score, работа с PostgreSQL
- `ml-service` — анализ аномалий и подозрительных отзывов
- `browser-extension` — popup-интерфейс для проверки товара по текущей вкладке
- `infra/postgres` — SQL-схема и seed-данные
- `docs` — архитектура и материалы по проекту
- `demo` — локальная demo-страница

## Что уже реализовано

- расчет `trust score` для продавца и товара
- поддержка реальных ссылок `OZON`, `Wildberries`, `Яндекс Маркета`
- fallback-анализ для неизвестных `product id`
- PostgreSQL-схема для маркетплейсов, продавцов, товаров, отзывов, метрик и истории анализа
- сохранение результатов проверок в PostgreSQL
- endpoint последних проверок: `GET /api/v1/checks/recent`
- персональная история проверок через `client_id`
- backend-side page enrichment по HTML страницы товара
- Redis-кеш с TTL 1 час
- браузерное расширение с кнопкой `Проверить товар`
- локальный dashboard для демонстрации проекта

## Быстрый запуск через Docker

```bash
docker compose up --build
```

После запуска будут доступны:

- backend API: `http://localhost:8080`
- ML service: `http://localhost:8001`
- health check: `http://localhost:8080/health`
- локальный dashboard: `http://localhost:8080/dashboard`

## Работа с локальным PostgreSQL

Если PostgreSQL уже установлен на компьютере:

1. Создай базу данных `seller_trust`
2. Выполни SQL-скрипт [init.sql](C:/Users/kiril/OneDrive/Документы/diplom/infra/postgres/init.sql) через pgAdmin4
3. Перед запуском backend задай переменную окружения:

```powershell
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/seller_trust?sslmode=disable"
```

В этом режиме backend будет:

- читать продавцов, товары и отзывы из PostgreSQL
- сохранять `analysis runs` и `trust snapshots`
- отдавать историю последних проверок

Если `DATABASE_URL` не задан или PostgreSQL недоступен, backend использует fallback-данные.

## Сценарий работы расширения

1. Открой `chrome://extensions/`
2. Включи режим разработчика
3. Нажми `Load unpacked`
4. Выбери папку [browser-extension](C:/Users/kiril/OneDrive/Документы/diplom/browser-extension)
5. Открой карточку товара на `OZON`, `Wildberries` или `Яндекс Маркете`
6. Нажми на иконку расширения
7. Нажми `Проверить товар`

Popup автоматически берет URL текущей вкладки и отправляет его в backend.

## Локальная демонстрация для защиты

Для простой локальной демонстрации:

1. Запусти проект через `docker compose up --build`
2. Открой `http://localhost:8080/dashboard`
3. Вставь реальную ссылку товара
4. Запусти анализ
5. Покажи:
   - итоговый `trust score`
   - достоверность рейтинга
   - причины риска
   - персональную историю проверок
   - сигналы страницы, извлеченные backend из HTML
   - сохранение результатов в backend-слой

## Примеры API

Проверка по ссылке товара:

```bash
curl -X POST http://localhost:8080/api/v1/trust/analyze-url \
  -H "Content-Type: application/json" \
  -d "{\"product_url\":\"https://www.ozon.ru/product/test-2530782332/\"}"
```

Последние проверки:

```bash
curl http://localhost:8080/api/v1/checks/recent
```

Сводка:

```bash
curl http://localhost:8080/api/v1/overview
```

## Основной сценарий системы

1. Пользователь открывает страницу товара на маркетплейсе
2. Расширение или dashboard получает URL товара
3. Go backend определяет маркетплейс и извлекает `product id`
4. Backend сначала пытается получить данные из PostgreSQL
5. Если товара нет в БД, используется fallback-модель по URL и `product id`
6. Backend пытается получить дополнительные сигналы из HTML страницы товара
7. Backend вызывает Python anti-fraud сервис
8. Backend рассчитывает итоговый `trust score`
9. Результат кешируется в Redis и сохраняется в PostgreSQL
10. История проверок может фильтроваться по `client_id`
11. Клиент получает trust score, authenticity score и причины риска

## Текущее состояние проекта

Сейчас это уже хороший локальный MVP для дипломного проекта:

- есть реальный URL-based сценарий проверки
- есть хранение данных и истории
- есть anti-fraud сервис
- есть браузерное расширение
- есть локальный сайт для демонстрации

## Дальнейшее развитие

Следующий крупный шаг:

- полноценные marketplace-specific парсеры для `OZON`, `WB`, `Яндекс Маркета`
- отдельная аналитика продавца
- более глубокий анализ отзывов и рейтингов
- расширение веб-интерфейса до полноценного личного кабинета
