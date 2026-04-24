export type SortOrder = "asc" | "desc";

export interface BaseResponse {
  message: string;
}

export interface HealthResponse {
  status: string;
  reason?: string;
}

export type MetricsSnapshot = Record<string, number>;

export interface PaginationMeta {
  page: number;
  pageSize: number;
  returnedItems: number;
  hasMore: boolean;
  sortBy: string;
  sortOrder: SortOrder;
}

export interface Wallet {
  id: string;
  owner: string;
  currency: string;
  createdAt: string;
  balance: number;
}

export interface WalletListResponse extends BaseResponse {
  wallets?: Wallet[];
  pagination?: PaginationMeta;
}

export interface WalletDetailResponse extends BaseResponse {
  wallet?: Wallet;
}

export interface WalletBalanceResponse extends BaseResponse {
  walletId: string;
  currency: string;
  balance: number;
}

export type TransactionType =
  | "OPENING_BALANCE"
  | "CREDIT"
  | "DEBIT"
  | "TRANSFER_IN"
  | "TRANSFER_OUT";

export interface Transaction {
  id: string;
  walletId: string;
  type: TransactionType;
  amount: number;
  counterpartyWalletId?: string;
  reference?: string;
  description?: string;
  balanceAfter: number;
  occurredAt: string;
  eventVersion: number;
}

export interface TransactionFilters {
  type?: string;
  occurredFrom?: string;
  occurredTo?: string;
}

export interface TransactionHistoryResponse extends BaseResponse {
  pagination?: PaginationMeta;
  filters?: TransactionFilters;
  transactions?: Transaction[];
}

export type LedgerEntryType = "DEBIT" | "CREDIT";

export interface LedgerEntry {
  id: string;
  movementId?: string;
  walletId: string;
  aggregateId: string;
  transactionId: string;
  eventId: string;
  eventType: string;
  eventVersion: number;
  transactionType: TransactionType;
  entryType: LedgerEntryType;
  amount: number;
  currency: string;
  counterpartyWalletId?: string;
  reference?: string;
  description?: string;
  occurredAt: string;
  createdAt: string;
}

export interface LedgerFilters {
  movementId?: string;
  walletId?: string;
  entryType?: string;
  eventType?: string;
  occurredFrom?: string;
  occurredTo?: string;
}

export interface LedgerEntryListResponse extends BaseResponse {
  pagination?: PaginationMeta;
  filters?: LedgerFilters;
  ledgerEntries?: LedgerEntry[];
}

export type LedgerMovementType = "OPENING_BALANCE" | "CREDIT" | "DEBIT" | "TRANSFER";
export type LedgerMovementStatus = "PENDING" | "COMPLETED" | "INCONSISTENT";

export interface LedgerMovement {
  id: string;
  movementType: LedgerMovementType;
  reference?: string;
  status: LedgerMovementStatus;
  currency: string;
  totalDebit: number;
  totalCredit: number;
  entryCount: number;
  sourceWalletId?: string;
  destinationWalletId?: string;
  aggregateId?: string;
  eventId?: string;
  eventType?: string;
  occurredAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface LedgerMovementFilters {
  walletId?: string;
  movementType?: string;
  status?: string;
  reference?: string;
  occurredFrom?: string;
  occurredTo?: string;
}

export interface LedgerMovementListResponse extends BaseResponse {
  pagination?: PaginationMeta;
  filters?: LedgerMovementFilters;
  ledgerMovements?: LedgerMovement[];
}

export interface LedgerMovementDetailResponse extends BaseResponse {
  ledgerMovement?: LedgerMovement;
}

export interface DeadLetterKafkaMeta {
  topic: string;
  partition: number;
  offset: number;
  consumerGroup: string;
}

export interface DeadLetter {
  deadLetterKey: string;
  eventId: string;
  eventType: string;
  aggregateId: string;
  status: string;
  failureKind: string;
  retryAttempts: number;
  lastError: string;
  payload: string;
  kafka: DeadLetterKafkaMeta;
  firstFailedAt: string;
  lastFailedAt: string;
  deadLetteredAt: string;
  reprocessedAt?: string;
  resolvedAt?: string;
}

export interface DeadLetterFilters {
  status?: string;
  eventType?: string;
  aggregateId?: string;
  failureKind?: string;
}

export interface DeadLetterListResponse extends BaseResponse {
  deadLetters?: DeadLetter[];
  pagination?: PaginationMeta;
  filters?: DeadLetterFilters;
}

export interface DeadLetterDetailResponse extends BaseResponse {
  deadLetter?: DeadLetter;
}

export interface DeadLetterReprocessResponse extends BaseResponse {
  deadLetter?: DeadLetter;
}

export interface CreateWalletRequest {
  owner: string;
  currency: string;
  openingBalance: number;
}

export interface CreateWalletResponse extends BaseResponse {
  id: string;
}

export interface CreditWalletRequest {
  amount: number;
  reference?: string;
  description?: string;
}

export interface DebitWalletRequest {
  amount: number;
  reference?: string;
  description?: string;
}

export interface TransferFundsRequest {
  destinationWalletId: string;
  amount: number;
  reference?: string;
  description?: string;
}

export interface ApiErrorShape {
  message?: string;
}
