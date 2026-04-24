"use client";

import Link from "next/link";

import { StatusBadge } from "@/components/ui/status-badge";
import { formatCurrency, formatDateTime, truncateMiddle } from "@/lib/format";
import { LedgerMovement } from "@/lib/types";

function movementTone(status: LedgerMovement["status"]) {
  switch (status) {
  case "COMPLETED":
    return "ready";
  case "PENDING":
    return "pending";
  default:
    return "down";
  }
}

export function LedgerMovementsTable({
  movements
}: {
  movements: LedgerMovement[];
}) {
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-left text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-slate-500">
            <th className="pb-3 font-medium">Occurred</th>
            <th className="pb-3 font-medium">Movement</th>
            <th className="pb-3 font-medium">Status</th>
            <th className="pb-3 font-medium">Debit</th>
            <th className="pb-3 font-medium">Credit</th>
            <th className="pb-3 font-medium">Wallets</th>
            <th className="pb-3 font-medium">Reference</th>
            <th className="pb-3 font-medium">Entries</th>
          </tr>
        </thead>
        <tbody>
          {movements.map((movement) => (
            <tr key={movement.id} className="border-b border-slate-100 last:border-b-0">
              <td className="py-4 text-slate-600">{formatDateTime(movement.occurredAt)}</td>
              <td className="py-4 font-medium text-slate-950">
                <Link href={`/ledger-movements/${movement.id}`} className="text-teal-700 hover:text-teal-900">
                  {movement.movementType}
                </Link>
              </td>
              <td className="py-4">
                <StatusBadge label={movement.status} tone={movementTone(movement.status)} />
              </td>
              <td className="py-4 text-slate-700">{formatCurrency(movement.totalDebit, movement.currency || "USD")}</td>
              <td className="py-4 text-slate-700">{formatCurrency(movement.totalCredit, movement.currency || "USD")}</td>
              <td className="py-4 text-slate-500">
                {movement.sourceWalletId ? truncateMiddle(movement.sourceWalletId) : "external"}
                {" -> "}
                {movement.destinationWalletId ? truncateMiddle(movement.destinationWalletId) : "external"}
              </td>
              <td className="py-4 text-slate-700">{movement.reference || "n/a"}</td>
              <td className="py-4 text-slate-500">{movement.entryCount}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
