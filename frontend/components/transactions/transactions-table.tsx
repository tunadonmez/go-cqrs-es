import { StatusBadge } from "@/components/ui/status-badge";
import { formatCurrency, formatDateTime, truncateMiddle } from "@/lib/format";
import { Transaction } from "@/lib/types";

export function TransactionsTable({
  transactions,
  currency
}: {
  transactions: Transaction[];
  currency: string;
}) {
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-left text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-slate-500">
            <th className="pb-3 font-medium">Occurred</th>
            <th className="pb-3 font-medium">Type</th>
            <th className="pb-3 font-medium">Amount</th>
            <th className="pb-3 font-medium">Balance After</th>
            <th className="pb-3 font-medium">Reference</th>
            <th className="pb-3 font-medium">Counterparty</th>
          </tr>
        </thead>
        <tbody>
          {transactions.map((transaction) => (
            <tr key={transaction.id} className="border-b border-slate-100 last:border-b-0">
              <td className="py-4 text-slate-600">{formatDateTime(transaction.occurredAt)}</td>
              <td className="py-4">
                <StatusBadge label={transaction.type} tone="ready" />
              </td>
              <td className="py-4 font-medium text-slate-950">
                {formatCurrency(transaction.amount, currency)}
              </td>
              <td className="py-4 text-slate-700">
                {formatCurrency(transaction.balanceAfter, currency)}
              </td>
              <td className="py-4 text-slate-700">{transaction.reference || "n/a"}</td>
              <td className="py-4 text-slate-500">
                {transaction.counterpartyWalletId ? truncateMiddle(transaction.counterpartyWalletId) : "n/a"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
