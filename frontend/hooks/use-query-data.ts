"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getDeadLetter,
  getQueryHealth,
  getQueryMetrics,
  getQueryReady,
  getWallet,
  getWalletBalance,
  getWalletTransactions,
  listDeadLetters,
  listWallets,
  reprocessDeadLetter
} from "@/lib/api/query-api";

export function useQueryHealth() {
  return useQuery({ queryKey: ["query-health"], queryFn: getQueryHealth });
}

export function useQueryReady() {
  return useQuery({ queryKey: ["query-ready"], queryFn: getQueryReady });
}

export function useQueryMetrics() {
  return useQuery({ queryKey: ["query-metrics"], queryFn: getQueryMetrics });
}

export function useWallets(params: Record<string, string | number | undefined>) {
  return useQuery({
    queryKey: ["wallets", params],
    queryFn: () => listWallets(params)
  });
}

export function useWallet(walletId: string) {
  return useQuery({
    queryKey: ["wallet", walletId],
    queryFn: () => getWallet(walletId),
    enabled: Boolean(walletId)
  });
}

export function useWalletBalance(walletId: string) {
  return useQuery({
    queryKey: ["wallet-balance", walletId],
    queryFn: () => getWalletBalance(walletId),
    enabled: Boolean(walletId)
  });
}

export function useWalletTransactions(
  walletId: string,
  params: Record<string, string | number | undefined>
) {
  return useQuery({
    queryKey: ["wallet-transactions", walletId, params],
    queryFn: () => getWalletTransactions(walletId, params),
    enabled: Boolean(walletId)
  });
}

export function useDeadLetters(params: Record<string, string | number | undefined>) {
  return useQuery({
    queryKey: ["dead-letters", params],
    queryFn: () => listDeadLetters(params)
  });
}

export function useDeadLetter(deadLetterKey: string) {
  return useQuery({
    queryKey: ["dead-letter", deadLetterKey],
    queryFn: () => getDeadLetter(deadLetterKey),
    enabled: Boolean(deadLetterKey)
  });
}

export function useReprocessDeadLetterMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: reprocessDeadLetter,
    onSuccess: (data, key) => {
      queryClient.invalidateQueries({ queryKey: ["dead-letters"] });
      queryClient.invalidateQueries({ queryKey: ["dead-letter", key] });
      queryClient.invalidateQueries({ queryKey: ["query-metrics"] });
      queryClient.invalidateQueries({ queryKey: ["wallets"] });
    }
  });
}
