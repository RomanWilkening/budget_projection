import type { Person, Account, Position, PositionSeparator, ProjectionResponse, BridgeAccount, BridgeStatus, RecurringAnalysis, Settings, Depot, ScenarioModification } from "./types";

const BASE = "/api";

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API ${res.status}: ${body}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

// Persons
export const getPersons = () => request<Person[]>(`${BASE}/persons`);
export const getPerson = (id: number) => request<Person>(`${BASE}/persons/${id}`);
export const createPerson = (data: { name: string }) =>
  request<Person>(`${BASE}/persons`, { method: "POST", body: JSON.stringify(data) });
export const updatePerson = (id: number, data: { name: string }) =>
  request<Person>(`${BASE}/persons/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const deletePerson = (id: number) =>
  request<void>(`${BASE}/persons/${id}`, { method: "DELETE" });
export const reorderPersons = (ids: number[]) =>
  request<void>(`${BASE}/persons`, { method: "PUT", body: JSON.stringify(ids) });

// Accounts
export const getAccounts = () => request<Account[]>(`${BASE}/accounts`);
export const getAccount = (id: number) => request<Account>(`${BASE}/accounts/${id}`);
export const createAccount = (data: { name: string; balance: number; currency: string; showInProjection?: boolean }) =>
  request<Account>(`${BASE}/accounts`, { method: "POST", body: JSON.stringify(data) });
export const updateAccount = (id: number, data: { name: string; balance: number; currency: string; showInProjection?: boolean }) =>
  request<Account>(`${BASE}/accounts/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const deleteAccount = (id: number) =>
  request<void>(`${BASE}/accounts/${id}`, { method: "DELETE" });
export const reorderAccounts = (ids: number[]) =>
  request<void>(`${BASE}/accounts`, { method: "PUT", body: JSON.stringify(ids) });
export const addAccountOwner = (accountId: number, personId: number) =>
  request<void>(`${BASE}/accounts/${accountId}/owners`, {
    method: "POST",
    body: JSON.stringify({ personId }),
  });
export const removeAccountOwner = (accountId: number, personId: number) =>
  request<void>(`${BASE}/accounts/${accountId}/owners/${personId}`, { method: "DELETE" });

// Positions
export const getPositions = () => request<Position[]>(`${BASE}/positions`);
export const getPosition = (id: number) => request<Position>(`${BASE}/positions/${id}`);
export const createPosition = (data: Partial<Position>) =>
  request<Position>(`${BASE}/positions`, { method: "POST", body: JSON.stringify(data) });
export const updatePosition = (id: number, data: Partial<Position>) =>
  request<Position>(`${BASE}/positions/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const deletePosition = (id: number) =>
  request<void>(`${BASE}/positions/${id}`, { method: "DELETE" });
export const reorderPositions = (items: { type: string; id: number }[]) =>
  request<void>(`${BASE}/positions`, { method: "PUT", body: JSON.stringify(items) });

// Position Separators
export const getSeparators = () => request<PositionSeparator[]>(`${BASE}/position-separators`);
export const createSeparator = (data: { name: string }) =>
  request<PositionSeparator>(`${BASE}/position-separators`, { method: "POST", body: JSON.stringify(data) });
export const updateSeparator = (id: number, data: { name: string }) =>
  request<PositionSeparator>(`${BASE}/position-separators/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const deleteSeparator = (id: number) =>
  request<void>(`${BASE}/position-separators/${id}`, { method: "DELETE" });

// Projection
export const getProjection = (params: {
  months?: number;
  startDate?: string;
  granularity?: string;
}) => {
  const query = new URLSearchParams();
  if (params.months) query.set("months", String(params.months));
  if (params.startDate) query.set("startDate", params.startDate);
  if (params.granularity) query.set("granularity", params.granularity);
  return request<ProjectionResponse>(`${BASE}/projection?${query.toString()}`);
};

export const getProjectionWithScenario = (params: {
  months?: number;
  startDate?: string;
  granularity?: string;
  scenario: ScenarioModification;
}) => {
  const query = new URLSearchParams();
  if (params.months) query.set("months", String(params.months));
  if (params.startDate) query.set("startDate", params.startDate);
  if (params.granularity) query.set("granularity", params.granularity);
  return request<ProjectionResponse>(`${BASE}/projection?${query.toString()}`, {
    method: "POST",
    body: JSON.stringify(params.scenario),
  });
};

// Banking Bridge
export const getBankingBridgeStatus = () =>
  request<BridgeStatus>(`${BASE}/banking-bridge/status`);
export const getBankingBridgeAccounts = () =>
  request<BridgeAccount[]>(`${BASE}/banking-bridge/accounts`);
export const linkBankingBridgeAccount = (accountId: number, bankingBridgeAccountId: number | null) =>
  request<Account>(`${BASE}/accounts/${accountId}/link-banking-bridge`, {
    method: "POST",
    body: JSON.stringify({ bankingBridgeAccountId }),
  });
export const syncAccountBalance = (accountId: number) =>
  request<{ account: Account; oldBalance: number; newBalance: number; lastUpdate: string }>(
    `${BASE}/accounts/${accountId}/sync-balance`,
    { method: "POST" }
  );
export const syncAllBalances = () =>
  request<{ synced: number; total: number; results: Array<{ accountId: number; accountName: string; oldBalance: number; newBalance: number; error?: string }> }>(
    `${BASE}/banking-bridge/sync-all-balances`,
    { method: "POST" }
  );
export const analyzeRecurringTransactions = (accountId: number, months?: number) => {
  const query = months ? `?months=${months}` : "";
  return request<RecurringAnalysis>(`${BASE}/accounts/${accountId}/recurring-transactions${query}`);
};

// Settings
export const getSettings = () =>
  request<Settings>(`${BASE}/settings`);
export const updateSettings = (data: Settings) =>
  request<Settings>(`${BASE}/settings`, { method: "PUT", body: JSON.stringify(data) });

// Depots
export const getDepots = () => request<Depot[]>(`${BASE}/depots`);
export const getDepot = (id: number) => request<Depot>(`${BASE}/depots/${id}`);
export const createDepot = (data: { name: string; interestRate: number; accountIds: number[] }) =>
  request<Depot>(`${BASE}/depots`, { method: "POST", body: JSON.stringify(data) });
export const updateDepot = (id: number, data: { name: string; interestRate: number; accountIds: number[] }) =>
  request<Depot>(`${BASE}/depots/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const deleteDepot = (id: number) =>
  request<void>(`${BASE}/depots/${id}`, { method: "DELETE" });
export const reorderDepots = (ids: number[]) =>
  request<void>(`${BASE}/depots`, { method: "PUT", body: JSON.stringify(ids) });
