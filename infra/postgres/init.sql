CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'marketplace_code') THEN
        CREATE TYPE marketplace_code AS ENUM ('ozon', 'wildberries', 'yandex_market', 'demo_market');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS marketplaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code marketplace_code NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    base_url VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sellers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    marketplace_id UUID NOT NULL REFERENCES marketplaces(id) ON DELETE RESTRICT,
    external_id VARCHAR(128) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    legal_name VARCHAR(255),
    inn VARCHAR(12),
    ogrn VARCHAR(15),
    registered_at TIMESTAMPTZ,
    rating NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count INT NOT NULL DEFAULT 0,
    total_orders INT NOT NULL DEFAULT 0,
    successful_orders INT NOT NULL DEFAULT 0,
    canceled_orders INT NOT NULL DEFAULT 0,
    complaints_count INT NOT NULL DEFAULT 0,
    returns_count INT NOT NULL DEFAULT 0,
    average_review_score NUMERIC(3,2) NOT NULL DEFAULT 0,
    is_official_store BOOLEAN NOT NULL DEFAULT FALSE,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (marketplace_id, external_id)
);

CREATE TABLE IF NOT EXISTS product_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(128) NOT NULL UNIQUE,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    marketplace_id UUID NOT NULL REFERENCES marketplaces(id) ON DELETE RESTRICT,
    seller_id UUID REFERENCES sellers(id) ON DELETE SET NULL,
    external_id VARCHAR(128) NOT NULL,
    sku VARCHAR(128),
    title VARCHAR(255) NOT NULL,
    brand VARCHAR(120),
    category_id UUID REFERENCES product_categories(id) ON DELETE SET NULL,
    current_price NUMERIC(12,2),
    old_price NUMERIC(12,2),
    currency_code CHAR(3) NOT NULL DEFAULT 'RUB',
    rating NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count INT NOT NULL DEFAULT 0,
    one_star_count INT NOT NULL DEFAULT 0,
    two_star_count INT NOT NULL DEFAULT 0,
    three_star_count INT NOT NULL DEFAULT 0,
    four_star_count INT NOT NULL DEFAULT 0,
    five_star_count INT NOT NULL DEFAULT 0,
    questions_count INT NOT NULL DEFAULT 0,
    favorites_count INT NOT NULL DEFAULT 0,
    purchase_count INT NOT NULL DEFAULT 0,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (marketplace_id, external_id)
);

CREATE TABLE IF NOT EXISTS product_urls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    canonical_url TEXT NOT NULL UNIQUE,
    url_hash VARCHAR(64),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    last_checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    seller_id UUID REFERENCES sellers(id) ON DELETE SET NULL,
    external_id VARCHAR(128) NOT NULL,
    author_name VARCHAR(255),
    rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    review_text TEXT NOT NULL,
    pros TEXT,
    cons TEXT,
    is_verified_purchase BOOLEAN NOT NULL DEFAULT FALSE,
    helpful_count INT NOT NULL DEFAULT 0,
    media_count INT NOT NULL DEFAULT 0,
    source_created_at TIMESTAMPTZ NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, external_id)
);

CREATE TABLE IF NOT EXISTS seller_metrics_daily (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID NOT NULL REFERENCES sellers(id) ON DELETE CASCADE,
    metric_date DATE NOT NULL,
    rating NUMERIC(3,2) NOT NULL,
    total_orders INT NOT NULL DEFAULT 0,
    successful_orders INT NOT NULL DEFAULT 0,
    complaints_count INT NOT NULL DEFAULT 0,
    returns_count INT NOT NULL DEFAULT 0,
    review_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (seller_id, metric_date)
);

CREATE TABLE IF NOT EXISTS product_metrics_daily (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    metric_date DATE NOT NULL,
    price NUMERIC(12,2),
    rating NUMERIC(3,2) NOT NULL,
    review_count INT NOT NULL DEFAULT 0,
    one_star_count INT NOT NULL DEFAULT 0,
    five_star_count INT NOT NULL DEFAULT 0,
    questions_count INT NOT NULL DEFAULT 0,
    favorites_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, metric_date)
);

CREATE TABLE IF NOT EXISTS analysis_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    seller_id UUID REFERENCES sellers(id) ON DELETE SET NULL,
    marketplace_id UUID REFERENCES marketplaces(id) ON DELETE SET NULL,
    source_url TEXT NOT NULL,
    parser_version VARCHAR(32) NOT NULL DEFAULT 'v1',
    model_version VARCHAR(32) NOT NULL DEFAULT 'v1',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    error_message TEXT
);

CREATE TABLE IF NOT EXISTS trust_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_run_id UUID REFERENCES analysis_runs(id) ON DELETE SET NULL,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    seller_id UUID REFERENCES sellers(id) ON DELETE SET NULL,
    trust_score SMALLINT NOT NULL CHECK (trust_score BETWEEN 0 AND 100),
    trust_level VARCHAR(16) NOT NULL,
    rating_authenticity SMALLINT NOT NULL CHECK (rating_authenticity BETWEEN 0 AND 100),
    fake_review_risk NUMERIC(5,4) NOT NULL DEFAULT 0,
    rating_manipulation_risk NUMERIC(5,4) NOT NULL DEFAULT 0,
    text_similarity_risk NUMERIC(5,4) NOT NULL DEFAULT 0,
    review_burst_risk NUMERIC(5,4) NOT NULL DEFAULT 0,
    suspicious BOOLEAN NOT NULL DEFAULT FALSE,
    recommendation TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS trust_reasons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id UUID NOT NULL REFERENCES trust_snapshots(id) ON DELETE CASCADE,
    reason_code VARCHAR(64) NOT NULL,
    reason_text TEXT NOT NULL,
    severity SMALLINT NOT NULL DEFAULT 1 CHECK (severity BETWEEN 1 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS browser_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_url TEXT NOT NULL,
    marketplace_id UUID REFERENCES marketplaces(id) ON DELETE SET NULL,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    seller_id UUID REFERENCES sellers(id) ON DELETE SET NULL,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    client_type VARCHAR(32) NOT NULL DEFAULT 'browser_extension',
    client_id VARCHAR(128),
    client_version VARCHAR(32),
    success BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_sellers_marketplace_external_id ON sellers (marketplace_id, external_id);
CREATE INDEX IF NOT EXISTS idx_products_marketplace_external_id ON products (marketplace_id, external_id);
CREATE INDEX IF NOT EXISTS idx_product_urls_product_id ON product_urls (product_id);
CREATE INDEX IF NOT EXISTS idx_reviews_product_id_created_at ON reviews (product_id, source_created_at DESC);
CREATE INDEX IF NOT EXISTS idx_seller_metrics_daily_seller_id_date ON seller_metrics_daily (seller_id, metric_date DESC);
CREATE INDEX IF NOT EXISTS idx_product_metrics_daily_product_id_date ON product_metrics_daily (product_id, metric_date DESC);
CREATE INDEX IF NOT EXISTS idx_trust_snapshots_product_id_created_at ON trust_snapshots (product_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_browser_checks_checked_at ON browser_checks (checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_browser_checks_client_id_checked_at ON browser_checks (client_id, checked_at DESC);

INSERT INTO marketplaces (code, name, base_url) VALUES
    ('ozon', 'OZON', 'https://www.ozon.ru'),
    ('wildberries', 'Wildberries', 'https://www.wildberries.ru'),
    ('yandex_market', 'Яндекс Маркет', 'https://market.yandex.ru'),
    ('demo_market', 'Demo Market', 'https://demo-market.local')
ON CONFLICT (code) DO NOTHING;

INSERT INTO product_categories (slug, title) VALUES
    ('electronics', 'Электроника'),
    ('smart-watch', 'Умные часы'),
    ('headphones', 'Наушники')
ON CONFLICT (slug) DO NOTHING;

WITH marketplace_rows AS (
    SELECT id, code FROM marketplaces
),
category_rows AS (
    SELECT id, slug FROM product_categories
),
seed_sellers AS (
    INSERT INTO sellers (
        marketplace_id, external_id, display_name, legal_name, inn, ogrn,
        rating, review_count, total_orders, successful_orders, canceled_orders,
        complaints_count, returns_count, average_review_score, is_official_store, is_verified
    )
    SELECT
        m.id,
        seed.external_id,
        seed.display_name,
        seed.legal_name,
        seed.inn,
        seed.ogrn,
        seed.rating,
        seed.review_count,
        seed.total_orders,
        seed.successful_orders,
        seed.canceled_orders,
        seed.complaints_count,
        seed.returns_count,
        seed.average_review_score,
        seed.is_official_store,
        seed.is_verified
    FROM (
        VALUES
            ('ozon', 'ozon-seller-1001', 'TechNova Ozon', 'ООО ТехНова', '7701234567', '1234567890123', 4.8, 3210, 5200, 4990, 41, 37, 82, 4.7, TRUE, TRUE),
            ('wildberries', 'wb-seller-2002', 'BestElectro WB', 'ООО БестЭлектро', '7802345678', '2234567890123', 4.1, 1180, 1400, 1180, 54, 96, 133, 4.0, FALSE, TRUE),
            ('yandex_market', 'ym-seller-3003', 'Market Gadget Hub', 'ООО Маркет Гаджет', '5403456789', '3234567890123', 4.6, 890, 2100, 1988, 22, 28, 49, 4.5, TRUE, TRUE)
    ) AS seed(
        marketplace_code, external_id, display_name, legal_name, inn, ogrn,
        rating, review_count, total_orders, successful_orders, canceled_orders,
        complaints_count, returns_count, average_review_score, is_official_store, is_verified
    )
    JOIN marketplace_rows m ON m.code::text = seed.marketplace_code
    ON CONFLICT (marketplace_id, external_id) DO NOTHING
    RETURNING id, marketplace_id, external_id
)
SELECT 1;

WITH marketplace_rows AS (
    SELECT id, code FROM marketplaces
),
category_rows AS (
    SELECT id, slug FROM product_categories
),
seller_rows AS (
    SELECT s.id, s.external_id, m.code
    FROM sellers s
    JOIN marketplaces m ON m.id = s.marketplace_id
),
seed_products AS (
    INSERT INTO products (
        marketplace_id, seller_id, external_id, sku, title, brand, category_id,
        current_price, old_price, currency_code, rating, review_count,
        one_star_count, two_star_count, three_star_count, four_star_count, five_star_count,
        questions_count, favorites_count, purchase_count, last_seen_at
    )
    SELECT
        m.id,
        s.id,
        seed.external_id,
        seed.sku,
        seed.title,
        seed.brand,
        c.id,
        seed.current_price,
        seed.old_price,
        'RUB',
        seed.rating,
        seed.review_count,
        seed.one_star_count,
        seed.two_star_count,
        seed.three_star_count,
        seed.four_star_count,
        seed.five_star_count,
        seed.questions_count,
        seed.favorites_count,
        seed.purchase_count,
        NOW()
    FROM (
        VALUES
            ('ozon', 'ozon-seller-1001', 'ozon-product-100100', 'OZN-100100', 'Беспроводные наушники TechNova X', 'TechNova', 'headphones', 7990, 10990, 4.9, 1830, 21, 18, 34, 107, 1650, 126, 4021, 6840),
            ('wildberries', 'wb-seller-2002', 'wb-product-200200', 'WB-200200', 'Smart Watch Lite', 'BestElectro', 'smart-watch', 4590, 6990, 4.8, 910, 74, 11, 9, 15, 801, 64, 2980, 4270),
            ('yandex_market', 'ym-seller-3003', 'ym-product-300300', 'YM-300300', 'Планшет Market Tab 11', 'Gadget Hub', 'electronics', 21990, 25990, 4.7, 640, 19, 13, 22, 74, 512, 41, 1770, 2510)
    ) AS seed(
        marketplace_code, seller_external_id, external_id, sku, title, brand, category_slug,
        current_price, old_price, rating, review_count,
        one_star_count, two_star_count, three_star_count, four_star_count, five_star_count,
        questions_count, favorites_count, purchase_count
    )
    JOIN marketplace_rows m ON m.code::text = seed.marketplace_code
    JOIN seller_rows s ON s.external_id = seed.seller_external_id AND s.code = m.code
    JOIN category_rows c ON c.slug = seed.category_slug
    ON CONFLICT (marketplace_id, external_id) DO NOTHING
    RETURNING id, external_id
)
SELECT 1;

INSERT INTO product_urls (product_id, canonical_url, url_hash, is_primary, last_checked_at)
SELECT
    p.id,
    seed.canonical_url,
    md5(seed.canonical_url),
    TRUE,
    NOW()
FROM (
    VALUES
        ('ozon-product-100100', 'https://www.ozon.ru/product/besprovodnye-naushniki-technova-x-100100/'),
        ('wb-product-200200', 'https://www.wildberries.ru/catalog/200200/detail.aspx'),
        ('ym-product-300300', 'https://market.yandex.ru/product--market-tab-11/300300')
) AS seed(product_external_id, canonical_url)
JOIN products p ON p.external_id = seed.product_external_id
ON CONFLICT (canonical_url) DO NOTHING;

INSERT INTO reviews (
    product_id, seller_id, external_id, author_name, rating, review_text,
    pros, cons, is_verified_purchase, helpful_count, media_count, source_created_at
)
SELECT
    p.id,
    p.seller_id,
    seed.external_id,
    seed.author_name,
    seed.rating,
    seed.review_text,
    seed.pros,
    seed.cons,
    seed.is_verified_purchase,
    seed.helpful_count,
    seed.media_count,
    NOW() - seed.days_ago * INTERVAL '1 day'
FROM (
    VALUES
        ('ozon-product-100100', 'oz-r-1', 'Ирина', 5, 'Отличный звук, упаковка целая, продавец надежный', 'Звук и автономность', 'Нет', TRUE, 12, 1, 12),
        ('ozon-product-100100', 'oz-r-2', 'Максим', 4, 'Наушники хорошие, но кейс быстро пачкается', 'Хорошее шумоподавление', 'Маркий кейс', TRUE, 5, 0, 10),
        ('wb-product-200200', 'wb-r-1', 'Покупатель', 5, 'Товар бомба лучший лучший лучший', 'Цена', 'Нет', FALSE, 1, 0, 2),
        ('wb-product-200200', 'wb-r-2', 'Покупатель', 5, 'Товар бомба лучший лучший лучший', 'Дизайн', 'Нет', FALSE, 0, 0, 2),
        ('wb-product-200200', 'wb-r-3', 'Антон', 1, 'Через неделю перестал включаться', 'Нет', 'Сломался быстро', TRUE, 8, 0, 1),
        ('ym-product-300300', 'ym-r-1', 'Ольга', 5, 'Планшет работает стабильно, доставка вовремя', 'Экран', 'Камера средняя', TRUE, 4, 0, 7)
) AS seed(
    product_external_id, external_id, author_name, rating, review_text,
    pros, cons, is_verified_purchase, helpful_count, media_count, days_ago
)
JOIN products p ON p.external_id = seed.product_external_id
ON CONFLICT (product_id, external_id) DO NOTHING;

INSERT INTO seller_metrics_daily (
    seller_id, metric_date, rating, total_orders, successful_orders, complaints_count, returns_count, review_count
)
SELECT
    s.id,
    CURRENT_DATE - seed.days_ago,
    seed.rating,
    seed.total_orders,
    seed.successful_orders,
    seed.complaints_count,
    seed.returns_count,
    seed.review_count
FROM (
    VALUES
        ('ozon-seller-1001', 2, 4.8, 5200, 4990, 37, 82, 3210),
        ('wb-seller-2002', 2, 4.1, 1400, 1180, 96, 133, 1180),
        ('ym-seller-3003', 2, 4.6, 2100, 1988, 28, 49, 890)
) AS seed(seller_external_id, days_ago, rating, total_orders, successful_orders, complaints_count, returns_count, review_count)
JOIN sellers s ON s.external_id = seed.seller_external_id
ON CONFLICT (seller_id, metric_date) DO NOTHING;

INSERT INTO product_metrics_daily (
    product_id, metric_date, price, rating, review_count, one_star_count, five_star_count, questions_count, favorites_count
)
SELECT
    p.id,
    CURRENT_DATE - seed.days_ago,
    seed.price,
    seed.rating,
    seed.review_count,
    seed.one_star_count,
    seed.five_star_count,
    seed.questions_count,
    seed.favorites_count
FROM (
    VALUES
        ('ozon-product-100100', 1, 7990, 4.9, 1830, 21, 1650, 126, 4021),
        ('wb-product-200200', 1, 4590, 4.8, 910, 74, 801, 64, 2980),
        ('ym-product-300300', 1, 21990, 4.7, 640, 19, 512, 41, 1770)
) AS seed(product_external_id, days_ago, price, rating, review_count, one_star_count, five_star_count, questions_count, favorites_count)
JOIN products p ON p.external_id = seed.product_external_id
ON CONFLICT (product_id, metric_date) DO NOTHING;
