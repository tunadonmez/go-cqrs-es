import { appConfig } from "@/lib/config";
import {
  BaseResponse,
  CreateWalletRequest,
  CreateWalletResponse,
  CreditWalletRequest,
  DebitWalletRequest,
  HealthResponse,
  MetricsSnapshot,
  TransferFundsRequest
} from "@/lib/types";

import { apiRequest } from "./http";

export async function getCommandHealth() {
  return apiRequest<HealthResponse>(`${appConfig.commandApiUrl}/health`);
}

export async function getCommandReady() {
  return apiRequest<HealthResponse>(`${appConfig.commandApiUrl}/ready`);
}

export async function getCommandMetrics() {
  return apiRequest<MetricsSnapshot>(`${appConfig.commandApiUrl}/metrics`);
}

export async function createWallet(payload: CreateWalletRequest) {
  return apiRequest<CreateWalletResponse>(`${appConfig.commandApiUrl}/api/v1/wallets`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export async function creditWallet(walletId: string, payload: CreditWalletRequest) {
  return apiRequest<BaseResponse>(`${appConfig.commandApiUrl}/api/v1/wallets/${walletId}/credit`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export async function debitWallet(walletId: string, payload: DebitWalletRequest) {
  return apiRequest<BaseResponse>(`${appConfig.commandApiUrl}/api/v1/wallets/${walletId}/debit`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export async function transferFunds(walletId: string, payload: TransferFundsRequest) {
  return apiRequest<BaseResponse>(`${appConfig.commandApiUrl}/api/v1/wallets/${walletId}/transfer`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}
