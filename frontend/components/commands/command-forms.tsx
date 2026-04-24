"use client";

import type { ReactNode } from "react";
import { useMemo, useState } from "react";

import {
  useCreateWalletMutation,
  useCreditWalletMutation,
  useDebitWalletMutation,
  useTransferFundsMutation
} from "@/hooks/use-command-data";
import { Wallet } from "@/lib/types";
import { Card } from "@/components/ui/card";

function FormNotice({
  state
}: {
  state?: { kind: "success" | "error"; message: string } | null;
}) {
  if (!state) {
    return (
      <p className="text-sm text-slate-500">
        Commands return immediately. The query-side read model may update a moment later.
      </p>
    );
  }

  return (
    <div
      className={`rounded-2xl px-4 py-3 text-sm ${
        state.kind === "success" ? "bg-emerald-100 text-emerald-900" : "bg-rose-100 text-rose-900"
      }`}
    >
      {state.message}
    </div>
  );
}

function Field({
  label,
  children
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <label className="block">
      <span className="mb-2 block text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">
        {label}
      </span>
      {children}
    </label>
  );
}

function inputClassName() {
  return "w-full rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-950 outline-none transition focus:border-teal-600 focus:bg-white";
}

export function CommandForms({ wallets }: { wallets: Wallet[] }) {
  const [createState, setCreateState] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [creditState, setCreditState] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [debitState, setDebitState] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [transferState, setTransferState] = useState<{ kind: "success" | "error"; message: string } | null>(null);

  const createMutation = useCreateWalletMutation();
  const creditMutation = useCreditWalletMutation();
  const debitMutation = useDebitWalletMutation();
  const transferMutation = useTransferFundsMutation();

  const walletOptions = useMemo(() => wallets.map((wallet) => ({ value: wallet.id, label: `${wallet.owner} (${wallet.currency})` })), [wallets]);

  return (
    <div className="grid gap-6 xl:grid-cols-2">
      <Card>
        <h2 className="text-xl font-semibold text-slate-950">Create Wallet</h2>
        <p className="mt-2 text-sm text-slate-500">Command-side write into MongoDB. Query-side wallet list may lag briefly.</p>
        <form
          className="mt-6 space-y-4"
          onSubmit={async (event) => {
            event.preventDefault();
            const form = event.currentTarget;
            const formData = new FormData(form);
            try {
              const result = await createMutation.mutateAsync({
                owner: String(formData.get("owner")),
                currency: String(formData.get("currency")).toUpperCase(),
                openingBalance: Number(formData.get("openingBalance") || 0)
              });
              setCreateState({
                kind: "success",
                message: `${result?.message ?? "Wallet created."} Command ID: ${result?.id ?? "n/a"}`
              });
              form.reset();
            } catch (error) {
              setCreateState({
                kind: "error",
                message: error instanceof Error ? error.message : "Wallet creation failed."
              });
            }
          }}
        >
          <Field label="Owner">
            <input name="owner" required className={inputClassName()} placeholder="Ada Lovelace" />
          </Field>
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Currency">
              <input name="currency" required className={inputClassName()} placeholder="USD" />
            </Field>
            <Field label="Opening Balance">
              <input
                name="openingBalance"
                type="number"
                step="0.01"
                defaultValue="0"
                className={inputClassName()}
              />
            </Field>
          </div>
          <button className="rounded-2xl bg-slate-950 px-4 py-3 text-sm font-medium text-white">
            Submit Create Command
          </button>
        </form>
        <div className="mt-4">
          <FormNotice state={createState} />
        </div>
      </Card>

      <CommandActionCard
        title="Credit Wallet"
        description="Adds funds on the command side."
        state={creditState}
        walletOptions={walletOptions}
        submitLabel="Submit Credit Command"
        onSubmit={async (formData) => {
          const walletId = String(formData.get("walletId"));
          const result = await creditMutation.mutateAsync({
            walletId,
            payload: {
              amount: Number(formData.get("amount")),
              reference: String(formData.get("reference") || ""),
              description: String(formData.get("description") || "")
            }
          });
          setCreditState({ kind: "success", message: result?.message ?? "Wallet credited." });
        }}
        onError={(message) => setCreditState({ kind: "error", message })}
      />

      <CommandActionCard
        title="Debit Wallet"
        description="Removes funds on the command side."
        state={debitState}
        walletOptions={walletOptions}
        submitLabel="Submit Debit Command"
        onSubmit={async (formData) => {
          const walletId = String(formData.get("walletId"));
          const result = await debitMutation.mutateAsync({
            walletId,
            payload: {
              amount: Number(formData.get("amount")),
              reference: String(formData.get("reference") || ""),
              description: String(formData.get("description") || "")
            }
          });
          setDebitState({ kind: "success", message: result?.message ?? "Wallet debited." });
        }}
        onError={(message) => setDebitState({ kind: "error", message })}
      />

      <Card>
        <h2 className="text-xl font-semibold text-slate-950">Transfer Funds</h2>
        <p className="mt-2 text-sm text-slate-500">
          One command, two aggregates. Read-side source and destination wallets can update asynchronously.
        </p>
        <form
          className="mt-6 space-y-4"
          onSubmit={async (event) => {
            event.preventDefault();
            const form = event.currentTarget;
            const formData = new FormData(form);
            try {
              const result = await transferMutation.mutateAsync({
                walletId: String(formData.get("sourceWalletId")),
                payload: {
                  destinationWalletId: String(formData.get("destinationWalletId")),
                  amount: Number(formData.get("amount")),
                  reference: String(formData.get("reference") || ""),
                  description: String(formData.get("description") || "")
                }
              });
              setTransferState({ kind: "success", message: result?.message ?? "Transfer requested." });
              form.reset();
            } catch (error) {
              setTransferState({
                kind: "error",
                message: error instanceof Error ? error.message : "Transfer failed."
              });
            }
          }}
        >
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Source Wallet">
              <select name="sourceWalletId" className={inputClassName()} required defaultValue="">
                <option value="" disabled>
                  Select a source wallet
                </option>
                {walletOptions.map((wallet) => (
                  <option key={wallet.value} value={wallet.value}>
                    {wallet.label}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Destination Wallet">
              <select name="destinationWalletId" className={inputClassName()} required defaultValue="">
                <option value="" disabled>
                  Select a destination wallet
                </option>
                {walletOptions.map((wallet) => (
                  <option key={wallet.value} value={wallet.value}>
                    {wallet.label}
                  </option>
                ))}
              </select>
            </Field>
          </div>
          <Field label="Amount">
            <input name="amount" type="number" step="0.01" required className={inputClassName()} />
          </Field>
          <Field label="Reference">
            <input name="reference" className={inputClassName()} placeholder="invoice-1024" />
          </Field>
          <Field label="Description">
            <textarea name="description" className={inputClassName()} rows={3} placeholder="Settlement transfer" />
          </Field>
          <button className="rounded-2xl bg-slate-950 px-4 py-3 text-sm font-medium text-white">
            Submit Transfer Command
          </button>
        </form>
        <div className="mt-4">
          <FormNotice state={transferState} />
        </div>
      </Card>
    </div>
  );
}

function CommandActionCard({
  title,
  description,
  state,
  walletOptions,
  submitLabel,
  onSubmit,
  onError
}: {
  title: string;
  description: string;
  state: { kind: "success" | "error"; message: string } | null;
  walletOptions: Array<{ value: string; label: string }>;
  submitLabel: string;
  onSubmit: (formData: FormData) => Promise<void>;
  onError: (message: string) => void;
}) {
  return (
    <Card>
      <h2 className="text-xl font-semibold text-slate-950">{title}</h2>
      <p className="mt-2 text-sm text-slate-500">{description}</p>
      <form
        className="mt-6 space-y-4"
        onSubmit={async (event) => {
          event.preventDefault();
          const form = event.currentTarget;
          const formData = new FormData(form);
          try {
            await onSubmit(formData);
            form.reset();
          } catch (error) {
            onError(error instanceof Error ? error.message : `${title} failed.`);
          }
        }}
      >
        <Field label="Wallet">
          <select name="walletId" className={inputClassName()} required defaultValue="">
            <option value="" disabled>
              Select a wallet
            </option>
            {walletOptions.map((wallet) => (
              <option key={wallet.value} value={wallet.value}>
                {wallet.label}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Amount">
          <input name="amount" type="number" step="0.01" required className={inputClassName()} />
        </Field>
        <Field label="Reference">
          <input name="reference" className={inputClassName()} placeholder="ops-adjustment" />
        </Field>
        <Field label="Description">
          <textarea name="description" className={inputClassName()} rows={3} placeholder="Describe why this command is being sent" />
        </Field>
        <button className="rounded-2xl bg-slate-950 px-4 py-3 text-sm font-medium text-white">
          {submitLabel}
        </button>
      </form>
      <div className="mt-4">
        <FormNotice state={state} />
      </div>
    </Card>
  );
}
