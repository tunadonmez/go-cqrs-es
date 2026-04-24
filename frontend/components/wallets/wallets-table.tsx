import Link from "next/link";

import { StatusBadge } from "@/components/ui/status-badge";
import { formatCurrency, formatDateTime, truncateMiddle } from "@/lib/format";
import { Wallet } from "@/lib/types";

export function WalletsTable({ wallets }: { wallets: Wallet[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-left text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-slate-500">
            <th className="pb-3 font-medium">Wallet</th>
            <th className="pb-3 font-medium">Owner</th>
            <th className="pb-3 font-medium">Currency</th>
            <th className="pb-3 font-medium">Balance</th>
            <th className="pb-3 font-medium">Created</th>
          </tr>
        </thead>
        <tbody>
          {wallets.map((wallet) => (
            <tr key={wallet.id} className="border-b border-slate-100 last:border-b-0">
              <td className="py-4">
                <Link href={`/wallets/${wallet.id}`} className="font-medium text-slate-950 hover:text-teal-700">
                  {truncateMiddle(wallet.id)}
                </Link>
              </td>
              <td className="py-4 text-slate-700">{wallet.owner}</td>
              <td className="py-4">
                <StatusBadge label={wallet.currency} tone="resolved" />
              </td>
              <td className="py-4 font-medium text-slate-950">
                {formatCurrency(wallet.balance, wallet.currency)}
              </td>
              <td className="py-4 text-slate-500">{formatDateTime(wallet.createdAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
