import { Card } from "@/components/ui/card";

export default function OperationsPage() {
  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-[0.26em] text-teal-700">Operations</p>
        <h2 className="mt-3 text-3xl font-semibold text-slate-950">What operators can do today</h2>
        <p className="mt-3 max-w-3xl text-sm leading-6 text-slate-600">
          This page documents the real operational controls that exist in the repository today. It does not invent background jobs, schedulers, or replay buttons that the backend does not actually implement.
        </p>
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <Card>
          <h3 className="text-xl font-semibold text-slate-950">Read-Model Replay</h3>
          <p className="mt-3 text-sm text-slate-600">
            Replay is CLI-based. It rebuilds PostgreSQL from MongoDB and deliberately bypasses Kafka.
          </p>
          <pre className="mt-4 overflow-x-auto rounded-2xl bg-slate-950 p-4 text-xs text-slate-100">{`cd wallet-query
POSTGRES_DSN="host=localhost user=postgres password=postgres dbname=walletLedger port=5432 sslmode=disable TimeZone=UTC" \\
MONGODB_URI="mongodb://root:root@localhost:27017/walletLedger?authSource=admin" \\
go run . --replay

go run . --replay --aggregate=<wallet-id>`}</pre>
        </Card>

        <Card>
          <h3 className="text-xl font-semibold text-slate-950">Dead-Letter Reprocessing</h3>
          <p className="mt-3 text-sm text-slate-600">
            Reprocessing reuses the same projection path as normal consumption. It remains safe because `applyIdempotent` still guards duplicate effects.
          </p>
          <pre className="mt-4 overflow-x-auto rounded-2xl bg-slate-950 p-4 text-xs text-slate-100">{`# CLI
cd wallet-query
POSTGRES_DSN="host=localhost user=postgres password=postgres dbname=walletLedger port=5432 sslmode=disable TimeZone=UTC" \\
go run . --reprocess-dead-letter=<dead-letter-key>

# HTTP
POST /api/v1/dead-letters/<dead-letter-key>/reprocess`}</pre>
        </Card>
      </div>

      <div className="grid gap-6 xl:grid-cols-3">
        <Card>
          <h3 className="text-lg font-semibold text-slate-950">Snapshots</h3>
          <p className="mt-3 text-sm text-slate-600">
            Snapshotting is a write-side optimization only. This UI surfaces the concept, but it does not add snapshot admin controls because the backend does not expose them.
          </p>
        </Card>
        <Card>
          <h3 className="text-lg font-semibold text-slate-950">Eventual Consistency</h3>
          <p className="mt-3 text-sm text-slate-600">
            Command pages call `wallet-cmd`; list and detail pages call `wallet-query`. Expect a short lag between a successful command and the read model reflecting it.
          </p>
        </Card>
        <Card>
          <h3 className="text-lg font-semibold text-slate-950">Health and Metrics</h3>
          <p className="mt-3 text-sm text-slate-600">
            Both services expose `/health`, `/ready`, and `/metrics`. The dashboard reads them separately so operators can see divergence between the two services.
          </p>
        </Card>
      </div>
    </div>
  );
}
