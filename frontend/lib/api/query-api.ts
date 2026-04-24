import { appConfig } from "@/lib/config";
import {
  DeadLetterDetailResponse,
  DeadLetterListResponse,
  DeadLetterReprocessResponse,
  HealthResponse,
  LedgerEntryListResponse,
  LedgerMovementDetailResponse,
  LedgerMovementListResponse,
  MetricsSnapshot,
  TransactionHistoryResponse,
  WalletBalanceResponse,
  WalletDetailResponse,
  WalletListResponse
} from "@/lib/types";

import { apiRequest } from "./http";

function withQuery(base: string, params: Record<string, string | number | undefined>) {
  const url = new URL(base);
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== "") {
      url.searchParams.set(key, String(value));
    }
  }
  return url.toString();
}

export async function getQueryHealth() {
  return apiRequest<HealthResponse>(`${appConfig.queryApiUrl}/health`);
}

export async function getQueryReady() {
  return apiRequest<HealthResponse>(`${appConfig.queryApiUrl}/ready`);
}

export async function getQueryMetrics() {
  return apiRequest<MetricsSnapshot>(`${appConfig.queryApiUrl}/metrics`);
}

export async function listWallets(params: Record<string, string | number | undefined>) {
  return apiRequest<WalletListResponse>(withQuery(`${appConfig.queryApiUrl}/api/v1/wallets`, params));
}

export async function getWallet(walletId: string) {
  return apiRequest<WalletDetailResponse>(`${appConfig.queryApiUrl}/api/v1/wallets/${walletId}`);
}

export async function getWalletBalance(walletId: string) {
  return apiRequest<WalletBalanceResponse>(`${appConfig.queryApiUrl}/api/v1/wallets/${walletId}/balance`);
}

export async function getWalletTransactions(
  walletId: string,
  params: Record<string, string | number | undefined>
) {
  return apiRequest<TransactionHistoryResponse>(
    withQuery(`${appConfig.queryApiUrl}/api/v1/wallets/${walletId}/transactions`, params)
  );
}

export async function listDeadLetters(params: Record<string, string | number | undefined>) {
  return apiRequest<DeadLetterListResponse>(
    withQuery(`${appConfig.queryApiUrl}/api/v1/dead-letters`, params)
  );
}

export async function listLedgerEntries(params: Record<string, string | number | undefined>) {
  return apiRequest<LedgerEntryListResponse>(
    withQuery(`${appConfig.queryApiUrl}/api/v1/ledger-entries`, params)
  );
}

export async function getWalletLedgerEntries(
  walletId: string,
  params: Record<string, string | number | undefined>
) {
  return apiRequest<LedgerEntryListResponse>(
    withQuery(`${appConfig.queryApiUrl}/api/v1/wallets/${walletId}/ledger-entries`, params)
  );
}

export async function listLedgerMovements(params: Record<string, string | number | undefined>) {
  return apiRequest<LedgerMovementListResponse>(
    withQuery(`${appConfig.queryApiUrl}/api/v1/ledger-movements`, params)
  );
}

export async function getWalletLedgerMovements(
  walletId: string,
  params: Record<string, string | number | undefined>
) {
  return apiRequest<LedgerMovementListResponse>(
    withQuery(`${appConfig.queryApiUrl}/api/v1/wallets/${walletId}/ledger-movements`, params)
  );
}

export async function getLedgerMovement(movementId: string) {
  return apiRequest<LedgerMovementDetailResponse>(
    `${appConfig.queryApiUrl}/api/v1/ledger-movements/${movementId}`
  );
}

export async function getDeadLetter(deadLetterKey: string) {
  return apiRequest<DeadLetterDetailResponse>(
    `${appConfig.queryApiUrl}/api/v1/dead-letters/${deadLetterKey}`
  );
}

export async function reprocessDeadLetter(deadLetterKey: string) {
  return apiRequest<DeadLetterReprocessResponse>(
    `${appConfig.queryApiUrl}/api/v1/dead-letters/${deadLetterKey}/reprocess`,
    { method: "POST" }
  );
}
