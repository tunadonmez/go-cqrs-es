import { Card } from "@/components/ui/card";
import { formatCurrency, formatDateTime } from "@/lib/format";
import { Wallet, WalletBalanceResponse } from "@/lib/types";

export function WalletSummary({
  wallet,
  balance
}: {
  wallet: Wallet;
  balance?: WalletBalanceResponse | null;
}) {
  return (
    <div className="grid gap-4 lg:grid-cols-4">
      <Card>
        <p className="text-xs uppercase tracking-[0.18em] text-slate-500">Wallet ID</p>
        <p className="mt-3 break-all text-sm font-medium text-slate-950">{wallet.id}</p>
      </Card>
      <Card>
        <p className="text-xs uppercase tracking-[0.18em] text-slate-500">Owner</p>
        <p className="mt-3 text-lg font-semibold text-slate-950">{wallet.owner}</p>
      </Card>
      <Card>
        <p className="text-xs uppercase tracking-[0.18em] text-slate-500">Query Balance</p>
        <p className="mt-3 text-2xl font-semibold text-slate-950">
          {formatCurrency(balance?.balance ?? wallet.balance, balance?.currency ?? wallet.currency)}
        </p>
      </Card>
      <Card>
        <p className="text-xs uppercase tracking-[0.18em] text-slate-500">Created</p>
        <p className="mt-3 text-sm font-medium text-slate-950">{formatDateTime(wallet.createdAt)}</p>
      </Card>
    </div>
  );
}
