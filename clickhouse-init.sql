CREATE DATABASE IF NOT EXISTS otel_logs;

CREATE TABLE IF NOT EXISTS otel_logs.logs
(
    Timestamp DateTime64(9) CODEC(Delta, ZSTD(1)),
    traceID String CODEC(ZSTD(1)),
    spanID String CODEC(ZSTD(1)),
    traceFlags UInt32 CODEC(ZSTD(1)),
    severityText String CODEC(ZSTD(1)),
    severityNumber UInt8 CODEC(ZSTD(1)),
    body String CODEC(ZSTD(1)),
    resource_attributes Map(String, String) CODEC(ZSTD(1)),
    attributes Map(String, String) CODEC(ZSTD(1)),
    INDEX idx_timestamp Timestamp TYPE bloom_filter GRANULARITY 3,
    INDEX idx_trace_id traceID TYPE bloom_filter GRANULARITY 3,
    INDEX idx_severity_number severityNumber TYPE minmax GRANULARITY 3
) ENGINE = MergeTree()
PARTITION BY toDate(Timestamp)
ORDER BY (Timestamp, severityNumber, traceID)
TTL toDate(Timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;

-- Создаем отдельную таблицу для метрик, если это понадобится
CREATE TABLE IF NOT EXISTS otel_logs.metrics
(
    Timestamp DateTime64(9) CODEC(Delta, ZSTD(1)),
    name String CODEC(ZSTD(1)),
    description String CODEC(ZSTD(1)),
    value Float64 CODEC(ZSTD(1)),
    resource_attributes Map(String, String) CODEC(ZSTD(1)),
    attributes Map(String, String) CODEC(ZSTD(1))
) ENGINE = MergeTree()
PARTITION BY toDate(Timestamp)
ORDER BY (Timestamp, name)
TTL toDate(Timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

-- Пользователь default уже имеет права, так как указаны в clickhouse-users.xml 