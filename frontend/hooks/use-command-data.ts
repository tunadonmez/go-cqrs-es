"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createWallet,
  creditWallet,
  debitWallet,
  getCommandHealth,
  getCommandMetrics,
  getCommandReady,
  transferFunds
} from "@/lib/api/command-api";

export function useCommandHealth() {
  return useQuery({ queryKey: ["command-health"], queryFn: getCommandHealth });
}

export function useCommandReady() {
  return useQuery({ queryKey: ["command-ready"], queryFn: getCommandReady });
}

export function useCommandMetrics() {
  return useQuery({ queryKey: ["command-metrics"], queryFn: getCommandMetrics });
}

export function useCreateWalletMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createWallet,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["wallets"] });
      queryClient.invalidateQueries({ queryKey: ["command-metrics"] });
      queryClient.invalidateQueries({ queryKey: ["query-metrics"] });
    }
  });
}

export function useCreditWalletMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ walletId, payload }: { walletId: string; payload: Parameters<typeof creditWallet>[1] }) =>
      creditWallet(walletId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["wallets"] });
      queryClient.invalidateQueries({ queryKey: ["command-metrics"] });
    }
  });
}

export function useDebitWalletMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ walletId, payload }: { walletId: string; payload: Parameters<typeof debitWallet>[1] }) =>
      debitWallet(walletId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["wallets"] });
      queryClient.invalidateQueries({ queryKey: ["command-metrics"] });
    }
  });
}

export function useTransferFundsMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      walletId,
      payload
    }: {
      walletId: string;
      payload: Parameters<typeof transferFunds>[1];
    }) => transferFunds(walletId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["wallets"] });
      queryClient.invalidateQueries({ queryKey: ["command-metrics"] });
    }
  });
}
