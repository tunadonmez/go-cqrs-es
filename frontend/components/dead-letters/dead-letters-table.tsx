"use client";

import { useState } from "react";

import { StatusBadge } from "@/components/ui/status-badge";
import { formatDateTime, truncateMiddle } from "@/lib/format";
import { DeadLetter } from "@/lib/types";

export function DeadLettersTable({
  deadLetters,
  onReprocess,
  reprocessingKey
}: {
  deadLetters: DeadLetter[];
  onReprocess: (deadLetterKey: string) => void;
  reprocessingKey?: string | null;
}) {
  const [expandedKey, setExpandedKey] = useState<string | null>(null);

  return (
    <div className="space-y-3">
      {deadLetters.map((deadLetter) => {
        const expanded = expandedKey === deadLetter.deadLetterKey;
        const isReprocessing = reprocessingKey === deadLetter.deadLetterKey;

        return (
          <div key={deadLetter.deadLetterKey} className="rounded-3xl border border-slate-200 bg-white/90 p-5 shadow-panel">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
                <div>
                  <p className="text-xs uppercase tracking-[0.16em] text-slate-500">Dead Letter Key</p>
                  <p className="mt-1 font-medium text-slate-950">{truncateMiddle(deadLetter.deadLetterKey)}</p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-[0.16em] text-slate-500">Event</p>
                  <p className="mt-1 font-medium text-slate-950">{deadLetter.eventType}</p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-[0.16em] text-slate-500">Aggregate</p>
                  <p className="mt-1 text-slate-700">{truncateMiddle(deadLetter.aggregateId)}</p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-[0.16em] text-slate-500">Status</p>
                  <div className="mt-1 flex gap-2">
                    <StatusBadge label={deadLetter.status} />
                    <StatusBadge label={deadLetter.failureKind} tone={deadLetter.failureKind} />
                  </div>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-[0.16em] text-slate-500">Last Failed</p>
                  <p className="mt-1 text-slate-700">{formatDateTime(deadLetter.lastFailedAt)}</p>
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                <button
                  onClick={() => setExpandedKey(expanded ? null : deadLetter.deadLetterKey)}
                  className="rounded-2xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700"
                >
                  {expanded ? "Hide Details" : "Show Details"}
                </button>
                <button
                  onClick={() => onReprocess(deadLetter.deadLetterKey)}
                  disabled={isReprocessing}
                  className="rounded-2xl bg-slate-950 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
                >
                  {isReprocessing ? "Reprocessing..." : "Reprocess"}
                </button>
              </div>
            </div>
            {expanded ? (
              <div className="mt-5 grid gap-4 border-t border-slate-200 pt-5 lg:grid-cols-[1.1fr_0.9fr]">
                <div>
                  <p className="text-xs uppercase tracking-[0.16em] text-slate-500">Original Payload</p>
                  <pre className="mt-2 overflow-x-auto rounded-2xl bg-slate-950 p-4 text-xs text-slate-100">
                    {deadLetter.payload}
                  </pre>
                </div>
                <div className="space-y-4">
                  <div className="rounded-2xl bg-slate-50 p-4">
                    <p className="text-xs uppercase tracking-[0.16em] text-slate-500">Last Error</p>
                    <p className="mt-2 text-sm text-slate-800">{deadLetter.lastError || "n/a"}</p>
                  </div>
                  <div className="rounded-2xl bg-slate-50 p-4 text-sm text-slate-700">
                    <p><span className="font-medium text-slate-900">Topic:</span> {deadLetter.kafka.topic}</p>
                    <p className="mt-1"><span className="font-medium text-slate-900">Partition:</span> {deadLetter.kafka.partition}</p>
                    <p className="mt-1"><span className="font-medium text-slate-900">Offset:</span> {deadLetter.kafka.offset}</p>
                    <p className="mt-1"><span className="font-medium text-slate-900">Consumer Group:</span> {deadLetter.kafka.consumerGroup}</p>
                    <p className="mt-3"><span className="font-medium text-slate-900">Retry Attempts:</span> {deadLetter.retryAttempts}</p>
                    <p className="mt-1"><span className="font-medium text-slate-900">Dead-Lettered:</span> {formatDateTime(deadLetter.deadLetteredAt)}</p>
                    <p className="mt-1"><span className="font-medium text-slate-900">Reprocessed:</span> {formatDateTime(deadLetter.reprocessedAt)}</p>
                    <p className="mt-1"><span className="font-medium text-slate-900">Resolved:</span> {formatDateTime(deadLetter.resolvedAt)}</p>
                  </div>
                </div>
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
