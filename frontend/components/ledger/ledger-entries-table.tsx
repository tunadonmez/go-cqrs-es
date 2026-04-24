import { StatusBadge } from "@/components/ui/status-badge";
import { formatCurrency, formatDateTime, truncateMiddle } from "@/lib/format";
import { LedgerEntry } from "@/lib/types";

export function LedgerEntriesTable({
  entries,
  fallbackCurrency
}: {
  entries: LedgerEntry[];
  fallbackCurrency?: string;
}) {
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-left text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-slate-500">
            <th className="pb-3 font-medium">Occurred</th>
            <th className="pb-3 font-medium">Entry</th>
            <th className="pb-3 font-medium">Amount</th>
            <th className="pb-3 font-medium">Wallet</th>
            <th className="pb-3 font-medium">Event</th>
            <th className="pb-3 font-medium">Reference</th>
            <th className="pb-3 font-medium">Counterparty</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => (
            <tr key={entry.id} className="border-b border-slate-100 last:border-b-0">
              <td className="py-4 text-slate-600">{formatDateTime(entry.occurredAt)}</td>
              <td className="py-4">
                <StatusBadge label={entry.entryType} tone={entry.entryType === "DEBIT" ? "pending" : "ready"} />
              </td>
              <td className="py-4 font-medium text-slate-950">
                {formatCurrency(entry.amount, entry.currency || fallbackCurrency || "USD")}
              </td>
              <td className="py-4 text-slate-700">{truncateMiddle(entry.walletId)}</td>
              <td className="py-4 text-slate-500">{entry.eventType}</td>
              <td className="py-4 text-slate-700">{entry.reference || "n/a"}</td>
              <td className="py-4 text-slate-500">
                {entry.counterpartyWalletId ? truncateMiddle(entry.counterpartyWalletId) : "n/a"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
