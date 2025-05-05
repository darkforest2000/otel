CREATE DATABASE IF NOT EXISTS otel_logs;

CREATE TABLE IF NOT EXISTS otel_logs.logs
(
    Timestamp DateTime64(9) CODEC(Delta, ZSTD(1)),
    TraceId String CODEC(ZSTD(1)),
    SpanId String CODEC(ZSTD(1)),
    TraceFlags UInt32 CODEC(ZSTD(1)),
    SeverityText String CODEC(ZSTD(1)),
    SeverityNumber UInt8 CODEC(ZSTD(1)),
    Body String CODEC(ZSTD(1)),
    ResourceAttributes Map(String, String) CODEC(ZSTD(1)),
    Attributes Map(String, String) CODEC(ZSTD(1)),
    ResourceSchemaUrl String CODEC(ZSTD(1)),
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion String CODEC(ZSTD(1)),
    ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ScopeDroppedAttrCount UInt32 CODEC(ZSTD(1)),
    ScopeSchemaUrl String CODEC(ZSTD(1)),
    ServiceName LowCardinality(String) CODEC(ZSTD(1)),
    LogAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    INDEX idx_timestamp Timestamp TYPE bloom_filter GRANULARITY 3,
    INDEX idx_trace_id TraceId TYPE bloom_filter GRANULARITY 3,
    INDEX idx_severity_number SeverityNumber TYPE minmax GRANULARITY 3
) ENGINE = MergeTree()
PARTITION BY toDate(Timestamp)
ORDER BY (Timestamp, SeverityNumber, TraceId)
TTL toDate(Timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;

-- Создаем отдельную таблицу для метрик, если это понадобится
CREATE TABLE IF NOT EXISTS otel_logs.metrics
(
    Timestamp DateTime64(9) CODEC(Delta, ZSTD(1)),
    Name String CODEC(ZSTD(1)),
    Description String CODEC(ZSTD(1)),
    Value Float64 CODEC(ZSTD(1)),
    ResourceAttributes Map(String, String) CODEC(ZSTD(1)),
    Attributes Map(String, String) CODEC(ZSTD(1))
) ENGINE = MergeTree()
PARTITION BY toDate(Timestamp)
ORDER BY (Timestamp, Name)
TTL toDate(Timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

-- Пользователь default уже имеет права, так как указаны в clickhouse-users.xml 