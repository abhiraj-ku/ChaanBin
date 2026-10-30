create table if not exists app_logs(
    id bigserial,
    ingested_at timestamptz not null default now(),
    service_name varchar(100) not null,
    severity varchar(20) not null,
    message text not null,
    embedding halfvec(768),
    primary key(id,ingested_at)
) partition by range (ingested_at);

-- Apply the HNSW vector index directly to the parent table

create index logs_embd_idx on app_logs
using HNSW(embedding halfvec_cosine_ops)
with (m =16 , ef_construction=64);

-- Add standard B-Tree indexes to speed up the strict metadata filtering in our hybrid search
create index logs_time_idx on app_logs(ingested_at);
create index logs_severity_idx on app_logs(severity);
