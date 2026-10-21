create table if not exists app_log(
    id bigserial primary key,
    ingested_at timestamptz not null default now(),
    service_name varchar(100) not null,
    severity varchar(20) not null,
    message text not null,
    embedding halfvec(1536)
);

-- Indexing for fast Approximate Nearest Neighbor (ANN) search
create index logs_embedding_idx on app_log using hnsw( embedding halfvec_cosine_ops)
with (m =16, ef_construction=64);