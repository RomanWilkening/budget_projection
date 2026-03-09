export interface Person {
  id: number;
  name: string;
  sortOrder: number;
  accounts?: Account[];
  createdAt: string;
  updatedAt: string;
}

export interface Account {
  id: number;
  name: string;
  balance: number;
  currency: string;
  sortOrder: number;
  showInProjection: boolean;
  bankingBridgeAccountId?: number | null;
  owners?: Person[];
  createdAt: string;
  updatedAt: string;
}

export type PositionType = "income" | "expense" | "transfer";

export type FrequencyType =
  | "daily"
  | "weekly"
  | "biweekly"
  | "monthly"
  | "quarterly"
  | "semi_annually"
  | "annually";

export type BusinessDayRule =
  | "exact"
  | "last_business_day_before"
  | "first_business_day_after"
  | "last_business_day_of_month";

export interface Position {
  id: number;
  name: string;
  type: PositionType;
  amount: number;
  accountId?: number;
  account?: Account;
  sourceAccountId?: number;
  sourceAccount?: Account;
  targetAccountId?: number;
  targetAccount?: Account;
  frequencyType: FrequencyType;
  interval: number;
  dayOfMonth?: number;
  monthOfYear?: number;
  dayOfWeek?: number;
  businessDayRule: BusinessDayRule;
  startDate: string;
  endDate?: string;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

export interface PositionSeparator {
  id: number;
  name: string;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

export interface ProjectionDataPoint {
  date: string;
  balance: number;
}

export interface AccountProjection {
  id: number;
  name: string;
  currency: string;
  dataPoints: ProjectionDataPoint[];
}

export interface ProjectionResponse {
  accounts: AccountProjection[];
  depots: DepotProjection[];
  totals: ProjectionDataPoint[];
}

export interface Depot {
  id: number;
  name: string;
  interestRate: number;
  sortOrder: number;
  accounts?: Account[];
  createdAt: string;
  updatedAt: string;
}

export interface DepotProjection {
  id: number;
  name: string;
  interestRate: number;
  dataPoints: ProjectionDataPoint[];
}

export interface BridgeAccount {
  id: number;
  name: string;
  account_number: string;
  iban: string;
  bic: string;
  account_type: string;
  bank: string;
  bank_code: string;
  balance: number;
  currency: string;
  last_update: string;
}

export interface BridgeStatus {
  configured: boolean;
  connected: boolean;
  url: string;
  error?: string;
}

export interface RecurringPattern {
  name: string;
  counterpartIban: string;
  description: string;
  averageAmount: number;
  medianAmount: number;
  lastAmount: number;
  minAmount: number;
  maxAmount: number;
  isExpense: boolean;
  frequency: string;
  dayOfMonth: number | null;
  occurrences: number;
  confidence: number;
  bookingText: string;
  matchingPositionId?: number;
  matchingPositionName?: string;
  suggestedAction: "create" | "update" | "none";
}

export interface RecurringAnalysis {
  accountId: number;
  accountName: string;
  analyzedFrom: string;
  analyzedTo: string;
  transactionCount: number;
  patterns: RecurringPattern[];
}

export type Settings = Record<string, string>;
