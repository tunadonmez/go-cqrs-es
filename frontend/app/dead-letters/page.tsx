"use client";

import { useState } from "react";

import { DeadLettersTable } from "@/components/dead-letters/dead-letters-table";
import { Card } from "@/components/ui/card";
import { EmptyState, ErrorState, LoadingState } from "@/components/ui/state";
import { useDeadLetters, useReprocessDeadLetterMutation } from "@/hooks/use-query-data";

export default function DeadLettersPage() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [sortBy, setSortBy] = useState("createdAt");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [status, setStatus] = useState("");
  const [eventType, setEventType] = useState("");
  const [aggregateId, setAggregateId] = useState("");
  const [failureKind, setFailureKind] = useState("");
  const [feedback, setFeedback] = useState<string | null>(null);

  const deadLetters = useDeadLetters({
    page,
    pageSize,
    sortBy,
    sortOrder,
    status,
    eventType,
    aggregateId,
    failureKind
  });
  const reprocessMutation = useReprocessDeadLetterMutation();

  if (deadLetters.isLoading) {
    return <LoadingState label="Loading dead letters..." />;
  }

  if (deadLetters.error) {
    return <ErrorState title="Could not load dead letters" body={deadLetters.error instanceof Error ? deadLetters.error.message : "Unknown dead-letter failure."} />;
  }

  const items = deadLetters.data?.deadLetters ?? [];
  return (
    <div className="space-y-6">
      <Card>
        <div className="grid gap-4 lg:grid-cols-4">
          <input
            value={status}
            onChange={(event) => {
              setPage(1);
              setStatus(event.target.value);
            }}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
            placeholder="status: pending"
          />
          <input
            value={eventType}
            onChange={(event) => {
              setPage(1);
              setEventType(event.target.value);
            }}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
            placeholder="eventType"
          />
          <input
            value={aggregateId}
            onChange={(event) => {
              setPage(1);
              setAggregateId(event.target.value);
            }}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
            placeholder="aggregateId"
          />
          <input
            value={failureKind}
            onChange={(event) => {
              setPage(1);
              setFailureKind(event.target.value);
            }}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
            placeholder="failureKind"
          />
        </div>
        <div className="mt-4 flex flex-wrap gap-3">
          <select value={sortBy} onChange={(event) => setSortBy(event.target.value)} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
            <option value="createdAt">Sort by createdAt</option>
            <option value="updatedAt">Sort by updatedAt</option>
          </select>
          <select value={sortOrder} onChange={(event) => setSortOrder(event.target.value as "asc" | "desc")} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
            <option value="desc">Descending</option>
            <option value="asc">Ascending</option>
          </select>
          <select value={pageSize} onChange={(event) => setPageSize(Number(event.target.value))} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
            <option value="10">10 rows</option>
            <option value="20">20 rows</option>
            <option value="50">50 rows</option>
          </select>
        </div>
      </Card>

      {feedback ? <Card className="bg-teal-50 text-sm text-teal-900">{feedback}</Card> : null}

      {items.length === 0 ? (
        <EmptyState
          title="No dead letters for the current filter set"
          body="This can mean the system is healthy or simply that your filters are too narrow."
        />
      ) : (
        <DeadLettersTable
          deadLetters={items}
          reprocessingKey={reprocessMutation.variables ?? null}
          onReprocess={async (deadLetterKey) => {
            try {
              const result = await reprocessMutation.mutateAsync(deadLetterKey);
              setFeedback(result?.message ?? `Reprocessed ${deadLetterKey}`);
            } catch (error) {
              setFeedback(error instanceof Error ? error.message : `Failed to reprocess ${deadLetterKey}`);
            }
          }}
        />
      )}

      <div className="flex items-center justify-between">
        <button onClick={() => setPage((current) => Math.max(1, current - 1))} className="rounded-2xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700">
          Previous
        </button>
        <p className="text-sm text-slate-500">Page {page}</p>
        <button
          onClick={() => {
            if (deadLetters.data?.pagination?.hasMore) {
              setPage((current) => current + 1);
            }
          }}
          disabled={!deadLetters.data?.pagination?.hasMore}
          className="rounded-2xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 disabled:opacity-50"
        >
          Next
        </button>
      </div>
    </div>
  );
}
