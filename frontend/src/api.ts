import type { Person, Account, Position, ProjectionResponse } from "./types";

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

// Accounts
export const getAccounts = () => request<Account[]>(`${BASE}/accounts`);
export const getAccount = (id: number) => request<Account>(`${BASE}/accounts/${id}`);
export const createAccount = (data: { name: string; balance: number; currency: string }) =>
  request<Account>(`${BASE}/accounts`, { method: "POST", body: JSON.stringify(data) });
export const updateAccount = (id: number, data: { name: string; balance: number; currency: string }) =>
  request<Account>(`${BASE}/accounts/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const deleteAccount = (id: number) =>
  request<void>(`${BASE}/accounts/${id}`, { method: "DELETE" });
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
