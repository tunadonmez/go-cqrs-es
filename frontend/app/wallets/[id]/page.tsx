"use client";

import Link from "next/link";
import { useParams, usePathname, useRouter, useSearchParams } from "next/navigation";

import { LedgerEntriesTable } from "@/components/ledger/ledger-entries-table";
import { TransactionsTable } from "@/components/transactions/transactions-table";
import { WalletSummary } from "@/components/wallets/wallet-summary";
import { Card } from "@/components/ui/card";
import { EmptyState, ErrorState, LoadingState } from "@/components/ui/state";
import { useWallet, useWalletBalance, useWalletLedgerEntries, useWalletTransactions } from "@/hooks/use-query-data";

export default function WalletDetailPage() {
  const params = useParams<{ id: string }>();
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const walletId = params.id;
  const page = Number(searchParams.get("page") || "1");
  const pageSize = Number(searchParams.get("pageSize") || "10");
  const sortBy = searchParams.get("sortBy") || "occurredAt";
  const sortOrder = (searchParams.get("sortOrder") || "desc") as "asc" | "desc";
  const type = searchParams.get("type") || "";
  const occurredFrom = searchParams.get("occurredFrom") || "";
  const occurredTo = searchParams.get("occurredTo") || "";

  const wallet = useWallet(walletId);
  const balance = useWalletBalance(walletId);
  const transactions = useWalletTransactions(walletId, {
    page,
    pageSize,
    sortBy,
    sortOrder,
    type,
    occurredFrom,
    occurredTo
  });
  const ledger = useWalletLedgerEntries(walletId, {
    page: 1,
    pageSize: 10,
    sortBy: "occurredAt",
    sortOrder: "desc"
  });

  if (wallet.isLoading || transactions.isLoading || ledger.isLoading) {
    return <LoadingState label="Loading wallet detail..." />;
  }

  if (wallet.error || transactions.error || ledger.error) {
    return (
      <ErrorState
        title="Could not load wallet detail"
        body={
          wallet.error instanceof Error
            ? wallet.error.message
            : transactions.error instanceof Error
              ? transactions.error.message
              : ledger.error instanceof Error
                ? ledger.error.message
              : "Unknown wallet detail failure."
        }
      />
    );
  }

  const walletData = wallet.data?.wallet;
  if (!walletData) {
    return (
      <EmptyState
        title="Wallet not found in the read model"
        body="The query API returned no wallet. The command may have failed, or the projection may not have caught up yet."
      />
    );
  }

  const transactionItems = transactions.data?.transactions ?? [];
  const ledgerItems = ledger.data?.ledgerEntries ?? [];

  return (
    <div className="space-y-6">
      <WalletSummary wallet={walletData} balance={balance.data ?? undefined} />

      <Card>
        <h2 className="text-xl font-semibold text-slate-950">Transaction History</h2>
        <p className="mt-2 text-sm text-slate-500">
          Filters map directly to the query API: type, date bounds, sort, and pagination.
        </p>
        <div className="mt-6 grid gap-4 md:grid-cols-5">
          <FilterLink label="All" href={`/wallets/${walletId}`} active={!type && !occurredFrom && !occurredTo} />
          <FilterLink
            label="Credits"
            href={`/wallets/${walletId}?type=CREDIT`}
            active={type === "CREDIT"}
          />
          <FilterLink
            label="Debits"
            href={`/wallets/${walletId}?type=DEBIT`}
            active={type === "DEBIT"}
          />
          <FilterLink
            label="Transfers In"
            href={`/wallets/${walletId}?type=TRANSFER_IN`}
            active={type === "TRANSFER_IN"}
          />
          <FilterLink
            label="Transfers Out"
            href={`/wallets/${walletId}?type=TRANSFER_OUT`}
            active={type === "TRANSFER_OUT"}
          />
        </div>
        <form
          className="mt-6 grid gap-4 lg:grid-cols-5"
          onSubmit={(event) => {
            event.preventDefault();
            const formData = new FormData(event.currentTarget);
            const params = new URLSearchParams();
            const nextType = String(formData.get("type") || "");
            const nextOccurredFrom = String(formData.get("occurredFrom") || "");
            const nextOccurredTo = String(formData.get("occurredTo") || "");
            const nextSortBy = String(formData.get("sortBy") || "occurredAt");
            const nextSortOrder = String(formData.get("sortOrder") || "desc");
            const nextPageSize = String(formData.get("pageSize") || "10");

            if (nextType) params.set("type", nextType);
            if (nextOccurredFrom) params.set("occurredFrom", nextOccurredFrom);
            if (nextOccurredTo) params.set("occurredTo", nextOccurredTo);
            params.set("sortBy", nextSortBy);
            params.set("sortOrder", nextSortOrder);
            params.set("pageSize", nextPageSize);
            router.push(`${pathname}?${params.toString()}`);
          }}
        >
          <select name="type" defaultValue={type} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
            <option value="">All types</option>
            <option value="OPENING_BALANCE">OPENING_BALANCE</option>
            <option value="CREDIT">CREDIT</option>
            <option value="DEBIT">DEBIT</option>
            <option value="TRANSFER_IN">TRANSFER_IN</option>
            <option value="TRANSFER_OUT">TRANSFER_OUT</option>
          </select>
          <input
            name="occurredFrom"
            type="date"
            defaultValue={occurredFrom}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
          />
          <input
            name="occurredTo"
            type="date"
            defaultValue={occurredTo}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
          />
          <div className="grid grid-cols-2 gap-4">
            <select name="sortBy" defaultValue={sortBy} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
              <option value="occurredAt">occurredAt</option>
              <option value="amount">amount</option>
              <option value="eventVersion">eventVersion</option>
            </select>
            <select name="sortOrder" defaultValue={sortOrder} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
              <option value="desc">desc</option>
              <option value="asc">asc</option>
            </select>
          </div>
          <div className="grid grid-cols-[1fr_auto] gap-3">
            <select name="pageSize" defaultValue={String(pageSize)} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
              <option value="10">10 rows</option>
              <option value="20">20 rows</option>
              <option value="50">50 rows</option>
            </select>
            <button className="rounded-2xl bg-slate-950 px-4 py-3 text-sm font-medium text-white">
              Apply
            </button>
          </div>
        </form>
        {transactionItems.length === 0 ? (
          <p className="mt-6 text-sm text-slate-500">No transactions returned for the current filter set.</p>
        ) : (
          <div className="mt-6">
            <TransactionsTable transactions={transactionItems} currency={walletData.currency} />
          </div>
        )}
        <div className="mt-6 flex items-center justify-between">
          <Link
            href={buildTransactionHref(pathname, {
              page: Math.max(1, page - 1),
              pageSize,
              sortBy,
              sortOrder,
              type,
              occurredFrom,
              occurredTo
            })}
            className="rounded-2xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700"
          >
            Previous
          </Link>
          <p className="text-sm text-slate-500">Page {page}</p>
          <Link
            href={buildTransactionHref(pathname, {
              page: page + 1,
              pageSize,
              sortBy,
              sortOrder,
              type,
              occurredFrom,
              occurredTo
            })}
            className={`rounded-2xl border px-4 py-2 text-sm font-medium ${
              transactions.data?.pagination?.hasMore
                ? "border-slate-200 text-slate-700"
                : "pointer-events-none border-slate-100 text-slate-300"
            }`}
          >
            Next
          </Link>
        </div>
      </Card>

      <Card>
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-slate-950">Wallet Ledger</h2>
            <p className="mt-1 text-sm text-slate-500">
              Debit/credit entries for this wallet only. This is a read-side accounting view.
            </p>
          </div>
          <Link href={`/ledger?walletId=${walletId}`} className="text-sm font-medium text-teal-700">
            Open full ledger
          </Link>
        </div>
        {ledgerItems.length === 0 ? (
          <p className="mt-6 text-sm text-slate-500">No ledger entries returned for this wallet yet.</p>
        ) : (
          <div className="mt-6">
            <LedgerEntriesTable entries={ledgerItems} fallbackCurrency={walletData.currency} />
          </div>
        )}
      </Card>
    </div>
  );
}

function FilterLink({
  href,
  label,
  active
}: {
  href: string;
  label: string;
  active: boolean;
}) {
  return (
    <a
      href={href}
      className={`rounded-2xl border px-4 py-3 text-sm font-medium ${
        active
          ? "border-slate-950 bg-slate-950 text-white"
          : "border-slate-200 bg-slate-50 text-slate-700"
      }`}
    >
      {label}
    </a>
  );
}

function buildTransactionHref(
  pathname: string,
  params: {
    page: number;
    pageSize: number;
    sortBy: string;
    sortOrder: "asc" | "desc";
    type: string;
    occurredFrom: string;
    occurredTo: string;
  }
) {
  const search = new URLSearchParams();
  search.set("page", String(params.page));
  search.set("pageSize", String(params.pageSize));
  search.set("sortBy", params.sortBy);
  search.set("sortOrder", params.sortOrder);
  if (params.type) search.set("type", params.type);
  if (params.occurredFrom) search.set("occurredFrom", params.occurredFrom);
  if (params.occurredTo) search.set("occurredTo", params.occurredTo);
  return `${pathname}?${search.toString()}`;
}
