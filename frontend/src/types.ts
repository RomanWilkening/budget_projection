export interface Person {
  id: number;
  name: string;
  accounts?: Account[];
  createdAt: string;
  updatedAt: string;
}

export interface Account {
  id: number;
  name: string;
  balance: number;
  currency: string;
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
  totals: ProjectionDataPoint[];
}
