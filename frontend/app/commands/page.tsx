"use client";

import { CommandForms } from "@/components/commands/command-forms";
import { ErrorState, LoadingState } from "@/components/ui/state";
import { useWallets } from "@/hooks/use-query-data";

export default function CommandsPage() {
  const wallets = useWallets({ page: 1, pageSize: 100, sortBy: "createdAt", sortOrder: "desc" });

  if (wallets.isLoading) {
    return <LoadingState label="Loading wallet options for command forms..." />;
  }

  if (wallets.error) {
    return (
      <ErrorState
        title="Could not load command helpers"
        body={wallets.error instanceof Error ? wallets.error.message : "Unknown command helper failure."}
      />
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-[0.26em] text-teal-700">Command Side</p>
        <h2 className="mt-3 text-3xl font-semibold text-slate-950">Issue commands against the write service</h2>
        <p className="mt-3 max-w-3xl text-sm leading-6 text-slate-600">
          These forms call `wallet-cmd`. The read model shown elsewhere comes from `wallet-query`, so use this page with the expectation of eventual consistency rather than immediate read-after-write guarantees.
        </p>
      </div>
      <CommandForms wallets={wallets.data?.wallets ?? []} />
    </div>
  );
}
