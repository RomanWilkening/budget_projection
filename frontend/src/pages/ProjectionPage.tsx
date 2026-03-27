import { useCallback, useEffect, useMemo, useState } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ReferenceLine,
} from "recharts";
import { getProjection, getProjectionWithScenario, getPersons, getAccounts, getDepots, getPositions, updatePosition, createPosition, deletePosition } from "../api";
import type { ProjectionResponse, Person, Account, Depot, Position, ScenarioModification } from "../types";

const COLORS = [
  "#3182ce",
  "#38a169",
  "#e53e3e",
  "#d69e2e",
  "#805ad5",
  "#dd6b20",
  "#319795",
  "#d53f8c",
];

const MONTH_OPTIONS = [
  { value: 1, label: "1 Monat" },
  { value: 3, label: "3 Monate" },
  { value: 6, label: "6 Monate" },
  { value: 12, label: "1 Jahr" },
  { value: 24, label: "2 Jahre" },
  { value: 60, label: "5 Jahre" },
];

const FREQUENCY_LABELS: Record<string, string> = {
  daily: "Täglich",
  weekly: "Wöchentlich",
  biweekly: "Alle 2 Wochen",
  monthly: "Monatlich",
  quarterly: "Quartalsweise",
  semi_annually: "Halbjährlich",
  annually: "Jährlich",
};

const TYPE_LABELS: Record<string, string> = {
  income: "Einnahme",
  expense: "Ausgabe",
  transfer: "Umbuchung",
};

function formatCurrency(value: number): string {
  return value.toLocaleString("de-DE", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

function formatDateLabel(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString("de-DE", { day: "2-digit", month: "2-digit", year: "2-digit" });
}

interface NewPositionForm {
  name: string;
  type: "income" | "expense" | "transfer";
  amount: string;
  accountId: string;
  sourceAccountId: string;
  targetAccountId: string;
  frequencyType: string;
  interval: string;
  dayOfMonth: string;
  businessDayRule: string;
  startDate: string;
  endDate: string;
  growthRate: string;
}

const emptyNewPosition: NewPositionForm = {
  name: "",
  type: "expense",
  amount: "",
  accountId: "",
  sourceAccountId: "",
  targetAccountId: "",
  frequencyType: "monthly",
  interval: "1",
  dayOfMonth: "1",
  businessDayRule: "exact",
  startDate: new Date().toISOString().split("T")[0],
  endDate: "",
  growthRate: "0",
};

export default function ProjectionPage() {
  const [months, setMonths] = useState(6);
  const [customYears, setCustomYears] = useState("");
  const [data, setData] = useState<ProjectionResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [persons, setPersons] = useState<Person[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [depots, setDepots] = useState<Depot[]>([]);
  const [positions, setPositions] = useState<Position[]>([]);
  const [selectedPersonIds, setSelectedPersonIds] = useState<Set<number>>(new Set());
  const [selectedAccountIds, setSelectedAccountIds] = useState<Set<number>>(new Set());
  const [selectedDepotIds, setSelectedDepotIds] = useState<Set<number>>(new Set());

  // Scenario state
  const [scenarioOpen, setScenarioOpen] = useState(false);
  const [removedPositionIds, setRemovedPositionIds] = useState<Set<number>>(new Set());
  const [modifiedAmounts, setModifiedAmounts] = useState<Map<number, number>>(new Map());
  const [modifiedGrowthRates, setModifiedGrowthRates] = useState<Map<number, number>>(new Map());
  const [newPositions, setNewPositions] = useState<Partial<Position>[]>([]);
  const [showNewPosForm, setShowNewPosForm] = useState(false);
  const [newPosForm, setNewPosForm] = useState<NewPositionForm>({ ...emptyNewPosition });
  const [applyingScenario, setApplyingScenario] = useState(false);

  // Inflation state
  const [inflationRate, setInflationRate] = useState(0);

  const today = useMemo(() => new Date().toISOString().split("T")[0], []);

  const hasScenarioChanges = removedPositionIds.size > 0 || modifiedAmounts.size > 0 || modifiedGrowthRates.size > 0 || newPositions.length > 0;

  // Build the scenario modification object
  const scenarioMod = useMemo((): ScenarioModification | null => {
    if (!hasScenarioChanges) return null;
    // Collect all position IDs that have any modification
    const modifiedIds = new Set([...modifiedAmounts.keys(), ...modifiedGrowthRates.keys()]);
    const modified: Position[] = [];
    for (const id of modifiedIds) {
      const orig = positions.find((p) => p.id === id);
      if (orig) {
        const pos = { ...orig };
        if (modifiedAmounts.has(id)) pos.amount = modifiedAmounts.get(id)!;
        if (modifiedGrowthRates.has(id)) pos.growthRate = modifiedGrowthRates.get(id)!;
        modified.push(pos);
      }
    }
    return {
      modifiedPositions: modified,
      removedPositionIds: Array.from(removedPositionIds),
      newPositions,
    };
  }, [hasScenarioChanges, modifiedAmounts, modifiedGrowthRates, removedPositionIds, newPositions, positions]);

  // Fetch persons, accounts, depots, and positions once on mount
  useEffect(() => {
    Promise.all([getPersons(), getAccounts(), getDepots(), getPositions()])
      .then(([p, a, d, pos]) => {
        setPersons(p);
        setAccounts(a);
        setDepots(d);
        setPositions(pos);
        setSelectedPersonIds(new Set(p.map((pr) => pr.id)));
        setSelectedAccountIds(new Set(a.filter((ac) => ac.showInProjection).map((ac) => ac.id)));
        setSelectedDepotIds(new Set(d.map((dp) => dp.id)));
      })
      .catch(() => { /* filter panel will simply not render */ });
  }, []);

  // Fetch projection data (with or without scenario)
  const fetchProjection = useCallback(() => {
    setLoading(true);
    setError("");

    const inflParam = inflationRate > 0 ? inflationRate : undefined;
    const promise = scenarioMod
      ? getProjectionWithScenario({ months, startDate: today, inflationRate: inflParam, scenario: scenarioMod })
      : getProjection({ months, startDate: today, inflationRate: inflParam });

    let cancelled = false;
    promise
      .then((result) => {
        if (!cancelled) setData(result);
      })
      .catch((e: Error) => {
        if (!cancelled) setError(e.message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, [months, today, scenarioMod, inflationRate]);

  useEffect(() => {
    return fetchProjection();
  }, [fetchProjection]);

  // Helper to get account name by ID
  const getAccountName = useCallback((id: number | undefined) => {
    if (!id) return "–";
    return accounts.find((a) => a.id === id)?.name ?? `Konto #${id}`;
  }, [accounts]);

  // Scenario: toggle position removal
  const toggleRemovePosition = (id: number) => {
    setRemovedPositionIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  // Scenario: modify position amount
  const setPositionAmount = (id: number, amount: number) => {
    setModifiedAmounts((prev) => {
      const next = new Map(prev);
      const orig = positions.find((p) => p.id === id);
      if (orig && orig.amount === amount) {
        next.delete(id);
      } else {
        next.set(id, amount);
      }
      return next;
    });
  };

  // Scenario: modify position growth rate
  const setPositionGrowthRate = (id: number, growthRate: number) => {
    setModifiedGrowthRates((prev) => {
      const next = new Map(prev);
      const orig = positions.find((p) => p.id === id);
      if (orig && orig.growthRate === growthRate) {
        next.delete(id);
      } else {
        next.set(id, growthRate);
      }
      return next;
    });
  };

  // Scenario: add new virtual position from form
  const handleAddNewPosition = (e: React.FormEvent) => {
    e.preventDefault();
    const amount = parseFloat(newPosForm.amount);
    if (!newPosForm.name || isNaN(amount) || amount <= 0) return;

    const pos: Partial<Position> = {
      name: newPosForm.name,
      type: newPosForm.type,
      amount,
      frequencyType: newPosForm.frequencyType as Position["frequencyType"],
      interval: parseInt(newPosForm.interval) || 1,
      dayOfMonth: newPosForm.dayOfMonth ? parseInt(newPosForm.dayOfMonth) : undefined,
      businessDayRule: newPosForm.businessDayRule as Position["businessDayRule"],
      startDate: newPosForm.startDate,
      endDate: newPosForm.endDate || undefined,
      growthRate: parseFloat(newPosForm.growthRate) || 0,
    };

    if (newPosForm.type === "transfer") {
      if (newPosForm.sourceAccountId) pos.sourceAccountId = parseInt(newPosForm.sourceAccountId);
      if (newPosForm.targetAccountId) pos.targetAccountId = parseInt(newPosForm.targetAccountId);
    } else {
      if (newPosForm.accountId) pos.accountId = parseInt(newPosForm.accountId);
    }

    setNewPositions((prev) => [...prev, pos]);
    setNewPosForm({ ...emptyNewPosition });
    setShowNewPosForm(false);
  };

  // Scenario: remove a virtual new position
  const removeNewPosition = (index: number) => {
    setNewPositions((prev) => prev.filter((_, i) => i !== index));
  };

  // Scenario: reset all changes
  const resetScenario = () => {
    setRemovedPositionIds(new Set());
    setModifiedAmounts(new Map());
    setModifiedGrowthRates(new Map());
    setNewPositions([]);
  };

  // Scenario: apply all changes to actual positions in the database
  const applyScenarioToPositions = async () => {
    if (!hasScenarioChanges) return;
    setApplyingScenario(true);
    try {
      // 1. Update modified positions (amounts and growth rates)
      const modifiedIds = new Set([...modifiedAmounts.keys(), ...modifiedGrowthRates.keys()]);
      for (const id of modifiedIds) {
        const orig = positions.find((p) => p.id === id);
        if (!orig) continue;
        const updates: Partial<Position> = {};
        if (modifiedAmounts.has(id)) updates.amount = modifiedAmounts.get(id)!;
        if (modifiedGrowthRates.has(id)) updates.growthRate = modifiedGrowthRates.get(id)!;
        await updatePosition(id, { ...orig, ...updates });
      }

      // 2. Delete removed positions
      for (const id of removedPositionIds) {
        await deletePosition(id);
      }

      // 3. Create new positions
      for (const pos of newPositions) {
        await createPosition(pos);
      }

      // Refresh positions and reset scenario
      const updatedPositions = await getPositions();
      setPositions(updatedPositions);
      resetScenario();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      setError("Fehler beim Übernehmen: " + msg);
    } finally {
      setApplyingScenario(false);
    }
  };

  // Build a map of account ID -> owner person IDs using the accounts data
  const accountOwnerMap = useMemo(() => {
    const m = new Map<number, number[]>();
    for (const acc of accounts) {
      m.set(acc.id, (acc.owners ?? []).map((o) => o.id));
    }
    return m;
  }, [accounts]);

  // Determine which account IDs are visible based on person + account filter
  const visibleAccountIds = useMemo(() => {
    if (!data) return new Set<number>();
    return new Set(
      data.accounts
        .filter((acc) => {
          if (!selectedAccountIds.has(acc.id)) return false;
          const ownerIds = accountOwnerMap.get(acc.id) ?? [];
          // Account with no owners passes person filter
          if (ownerIds.length === 0) return true;
          // At least one owner must be selected
          return ownerIds.some((oid) => selectedPersonIds.has(oid));
        })
        .map((acc) => acc.id)
    );
  }, [data, selectedAccountIds, selectedPersonIds, accountOwnerMap]);

  const togglePerson = (id: number) => {
    setSelectedPersonIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleAccount = (id: number) => {
    setSelectedAccountIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleDepot = (id: number) => {
    setSelectedDepotIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  // Filtered accounts from projection data
  const filteredAccounts = data
    ? data.accounts.filter((acc) => visibleAccountIds.has(acc.id))
    : [];

  // Filtered depots from projection data
  const filteredDepots = data?.depots
    ? data.depots.filter((dp) => selectedDepotIds.has(dp.id))
    : [];

  // Build chart data: merge visible accounts into rows keyed by date
  const chartData = data
    ? data.totals.map((t, i) => {
        const row: Record<string, string | number> = {
          date: t.date,
          dateLabel: formatDateLabel(t.date),
        };
        let total = 0;
        for (const acc of filteredAccounts) {
          const bal = acc.dataPoints[i]?.balance ?? 0;
          row[acc.name] = bal;
          total += bal;
          // Add inflation-adjusted account line
          if (acc.inflationAdjustedDataPoints && acc.inflationAdjustedDataPoints[i]) {
            row[`${acc.name} (real)`] = acc.inflationAdjustedDataPoints[i].balance;
          }
        }
        row["Gesamt"] = total;
        // Add depot lines
        for (const dp of filteredDepots) {
          const bal = dp.dataPoints[i]?.balance ?? 0;
          row[`📊 ${dp.name}`] = bal;
          // Add inflation-adjusted depot line
          if (dp.inflationAdjustedDataPoints && dp.inflationAdjustedDataPoints[i]) {
            row[`📊 ${dp.name} (real)`] = dp.inflationAdjustedDataPoints[i].balance;
          }
        }
        // Add inflation-adjusted total line
        if (data.inflationAdjustedTotals && data.inflationAdjustedTotals[i]) {
          row["Gesamt (inflationsbereinigt)"] = data.inflationAdjustedTotals[i].balance;
        }
        return row;
      })
    : [];

  // Compute summary statistics for visible accounts only
  const hasInflation = data?.inflationAdjustedTotals && data.inflationAdjustedTotals.length > 0;
  const summaryStats = filteredAccounts.map((acc) => {
    const balances = acc.dataPoints.map((dp) => dp.balance);
    const startBal = balances[0] ?? 0;
    const endBal = balances[balances.length - 1] ?? 0;
    const adjEndBal = acc.inflationAdjustedDataPoints && acc.inflationAdjustedDataPoints.length > 0
      ? acc.inflationAdjustedDataPoints[acc.inflationAdjustedDataPoints.length - 1]?.balance ?? null
      : null;
    return {
      name: acc.name,
      currency: acc.currency,
      startBalance: startBal,
      endBalance: endBal,
      adjustedEndBalance: adjEndBal,
      change: endBal - startBal,
      monthlyNetFlow: acc.monthlyNetFlow,
      min: balances.length > 0 ? Math.min(...balances) : 0,
      max: balances.length > 0 ? Math.max(...balances) : 0,
    };
  });

  const totalStart = summaryStats.reduce((s, a) => s + a.startBalance, 0);
  const totalEnd = summaryStats.reduce((s, a) => s + a.endBalance, 0);
  const totalChange = totalEnd - totalStart;
  const totalMonthlyNetFlow = summaryStats.reduce((s, a) => s + a.monthlyNetFlow, 0);

  // Determine a good tick interval for X axis
  const tickInterval = chartData.length > 60 ? Math.floor(chartData.length / 12) : chartData.length > 30 ? Math.floor(chartData.length / 8) : undefined;

  return (
    <>
      <div className="page-header">
        <h2>Projektion</h2>
        <div className="projection-controls">
          {MONTH_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              className={`btn btn-sm ${months === opt.value && customYears === "" ? "btn-primary" : "btn-secondary"}`}
              onClick={() => { setMonths(opt.value); setCustomYears(""); }}
            >
              {opt.label}
            </button>
          ))}
          <div className="custom-years-input">
            <input
              type="number"
              min="1"
              max="50"
              placeholder="z.B. 10"
              value={customYears}
              onChange={(e) => {
                const val = e.target.value;
                setCustomYears(val);
                if (val === "") {
                  setMonths(6);
                  return;
                }
                const num = parseInt(val, 10);
                if (num >= 1 && num <= 50) {
                  setMonths(num * 12);
                }
              }}
              className="input-sm"
            />
            <span className="custom-years-label">Jahre</span>
          </div>
          <div className="custom-years-input">
            <input
              type="number"
              min="0"
              max="100"
              step="0.1"
              placeholder="z.B. 2.0"
              value={inflationRate || ""}
              onChange={(e) => {
                const val = e.target.value;
                if (val === "") {
                  setInflationRate(0);
                  return;
                }
                const num = parseFloat(val);
                if (!isNaN(num) && num >= 0 && num <= 100) {
                  setInflationRate(num);
                }
              }}
              className="input-sm"
            />
            <span className="custom-years-label">% Inflation</span>
          </div>
        </div>
      </div>

      {error && <div className="error-banner">{error}</div>}
      {loading && <div className="loading">Laden…</div>}

      {!loading && data && (
        <>
          {/* Filter Panel */}
          {(persons.length > 0 || accounts.length > 0 || depots.length > 0) && (
            <div className="card projection-filter-card">
              <h3>Filter</h3>
              <div className="projection-filter-groups">
                {persons.length > 0 && (
                  <div className="projection-filter-group">
                    <span className="projection-filter-label">Personen</span>
                    <div className="projection-filter-options">
                      {persons.map((p) => (
                        <label key={p.id} className="projection-filter-option">
                          <input
                            type="checkbox"
                            checked={selectedPersonIds.has(p.id)}
                            onChange={() => togglePerson(p.id)}
                          />
                          {p.name}
                        </label>
                      ))}
                    </div>
                  </div>
                )}
                {accounts.length > 0 && (
                  <div className="projection-filter-group">
                    <span className="projection-filter-label">Konten</span>
                    <div className="projection-filter-options">
                      {accounts.map((a) => (
                        <label key={a.id} className="projection-filter-option">
                          <input
                            type="checkbox"
                            checked={selectedAccountIds.has(a.id)}
                            onChange={() => toggleAccount(a.id)}
                          />
                          {a.name}
                        </label>
                      ))}
                    </div>
                  </div>
                )}
                {depots.length > 0 && (
                  <div className="projection-filter-group">
                    <span className="projection-filter-label">Depots</span>
                    <div className="projection-filter-options">
                      {depots.map((d) => (
                        <label key={d.id} className="projection-filter-option">
                          <input
                            type="checkbox"
                            checked={selectedDepotIds.has(d.id)}
                            onChange={() => toggleDepot(d.id)}
                          />
                          {d.name} ({d.interestRate.toFixed(2)}% p.a.)
                        </label>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Scenario Panel */}
          <div className={`card scenario-card ${hasScenarioChanges ? "scenario-active" : ""}`}>
            <div className="scenario-header" onClick={() => setScenarioOpen(!scenarioOpen)}>
              <h3>
                🔬 Szenario
                {hasScenarioChanges && <span className="scenario-badge">Aktiv</span>}
              </h3>
              <span className="scenario-toggle">{scenarioOpen ? "▲" : "▼"}</span>
            </div>

            {scenarioOpen && (
              <div className="scenario-content">
                <p className="scenario-hint">
                  Passen Sie Positionen temporär an, um verschiedene Szenarien zu simulieren.
                  Mit „Änderungen übernehmen" können Sie die Änderungen in die Positionen speichern.
                </p>

                {hasScenarioChanges && (
                  <div className="scenario-actions">
                    <button className="btn btn-sm btn-secondary" onClick={resetScenario}>
                      Szenario zurücksetzen
                    </button>
                    <button
                      className="btn btn-sm btn-primary"
                      onClick={applyScenarioToPositions}
                      disabled={applyingScenario}
                    >
                      {applyingScenario ? "Wird übernommen…" : "Änderungen übernehmen"}
                    </button>
                  </div>
                )}

                {/* Existing positions */}
                {positions.length > 0 && (
                  <div className="scenario-positions">
                    <table className="scenario-table">
                      <thead>
                        <tr>
                          <th>Aktiv</th>
                          <th>Position</th>
                          <th>Typ</th>
                          <th>Konto</th>
                          <th>Frequenz</th>
                          <th>Betrag (€)</th>
                          <th>Dynamik (% p.a.)</th>
                        </tr>
                      </thead>
                      <tbody>
                        {positions.map((pos) => {
                          const isRemoved = removedPositionIds.has(pos.id);
                          const modAmount = modifiedAmounts.get(pos.id);
                          const modGrowth = modifiedGrowthRates.get(pos.id);
                          const isModified = modAmount !== undefined || modGrowth !== undefined;
                          const accountName = pos.type === "transfer"
                            ? `${getAccountName(pos.sourceAccountId)} → ${getAccountName(pos.targetAccountId)}`
                            : getAccountName(pos.accountId);
                          return (
                            <tr key={pos.id} className={isRemoved ? "scenario-row-removed" : isModified ? "scenario-row-modified" : ""}>
                              <td>
                                <input
                                  type="checkbox"
                                  checked={!isRemoved}
                                  onChange={() => toggleRemovePosition(pos.id)}
                                />
                              </td>
                              <td className={isRemoved ? "text-strikethrough" : ""}>{pos.name}</td>
                              <td>
                                <span className={`type-badge type-${pos.type}`}>
                                  {TYPE_LABELS[pos.type] ?? pos.type}
                                </span>
                              </td>
                              <td>{accountName}</td>
                              <td>{FREQUENCY_LABELS[pos.frequencyType] ?? pos.frequencyType}</td>
                              <td>
                                <input
                                  type="number"
                                  className="scenario-amount-input"
                                  value={modAmount !== undefined ? modAmount : pos.amount}
                                  step="0.01"
                                  min="0"
                                  disabled={isRemoved}
                                  onChange={(e) => {
                                    const val = parseFloat(e.target.value);
                                    if (!isNaN(val)) setPositionAmount(pos.id, val);
                                  }}
                                />
                              </td>
                              <td>
                                <input
                                  type="number"
                                  className="scenario-amount-input"
                                  value={modGrowth !== undefined ? modGrowth : pos.growthRate}
                                  step="0.1"
                                  disabled={isRemoved}
                                  onChange={(e) => {
                                    const val = parseFloat(e.target.value);
                                    if (!isNaN(val)) setPositionGrowthRate(pos.id, val);
                                  }}
                                />
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )}

                {/* New virtual positions */}
                {newPositions.length > 0 && (
                  <div className="scenario-new-positions">
                    <h4>Neue Positionen (virtuell)</h4>
                    <table className="scenario-table">
                      <thead>
                        <tr>
                          <th></th>
                          <th>Position</th>
                          <th>Typ</th>
                          <th>Konto</th>
                          <th>Frequenz</th>
                          <th>Betrag (€)</th>
                          <th>Dynamik (% p.a.)</th>
                        </tr>
                      </thead>
                      <tbody>
                        {newPositions.map((pos, i) => {
                          const accountName = pos.type === "transfer"
                            ? `${getAccountName(pos.sourceAccountId)} → ${getAccountName(pos.targetAccountId)}`
                            : getAccountName(pos.accountId);
                          return (
                            <tr key={i} className="scenario-row-new">
                              <td>
                                <button
                                  className="btn btn-sm btn-icon btn-danger"
                                  onClick={() => removeNewPosition(i)}
                                  title="Entfernen"
                                >
                                  ✕
                                </button>
                              </td>
                              <td>{pos.name}</td>
                              <td>
                                <span className={`type-badge type-${pos.type}`}>
                                  {TYPE_LABELS[pos.type ?? ""] ?? pos.type}
                                </span>
                              </td>
                              <td>{accountName}</td>
                              <td>{FREQUENCY_LABELS[pos.frequencyType ?? ""] ?? pos.frequencyType}</td>
                              <td className="amount">{formatCurrency(pos.amount ?? 0)}</td>
                              <td className="amount">{pos.growthRate ? `${pos.growthRate.toFixed(1)}` : "0.0"}</td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )}

                {/* Add new position form */}
                {showNewPosForm ? (
                  <div className="scenario-new-form">
                    <h4>Neue virtuelle Position</h4>
                    <form onSubmit={handleAddNewPosition}>
                      <div className="form-row">
                        <div className="form-group">
                          <label>Name</label>
                          <input
                            type="text"
                            value={newPosForm.name}
                            onChange={(e) => setNewPosForm({ ...newPosForm, name: e.target.value })}
                            required
                            placeholder="Positionsname"
                          />
                        </div>
                        <div className="form-group">
                          <label>Typ</label>
                          <select
                            value={newPosForm.type}
                            onChange={(e) => setNewPosForm({ ...newPosForm, type: e.target.value as NewPositionForm["type"] })}
                          >
                            <option value="income">Einnahme</option>
                            <option value="expense">Ausgabe</option>
                            <option value="transfer">Umbuchung</option>
                          </select>
                        </div>
                      </div>
                      <div className="form-row">
                        <div className="form-group">
                          <label>Betrag (€)</label>
                          <input
                            type="number"
                            step="0.01"
                            min="0"
                            value={newPosForm.amount}
                            onChange={(e) => setNewPosForm({ ...newPosForm, amount: e.target.value })}
                            required
                          />
                        </div>
                        {newPosForm.type === "transfer" ? (
                          <>
                            <div className="form-group">
                              <label>Quellkonto</label>
                              <select
                                value={newPosForm.sourceAccountId}
                                onChange={(e) => setNewPosForm({ ...newPosForm, sourceAccountId: e.target.value })}
                              >
                                <option value="">– wählen –</option>
                                {accounts.map((a) => (
                                  <option key={a.id} value={a.id}>{a.name}</option>
                                ))}
                              </select>
                            </div>
                            <div className="form-group">
                              <label>Zielkonto</label>
                              <select
                                value={newPosForm.targetAccountId}
                                onChange={(e) => setNewPosForm({ ...newPosForm, targetAccountId: e.target.value })}
                              >
                                <option value="">– wählen –</option>
                                {accounts.map((a) => (
                                  <option key={a.id} value={a.id}>{a.name}</option>
                                ))}
                              </select>
                            </div>
                          </>
                        ) : (
                          <div className="form-group">
                            <label>Konto</label>
                            <select
                              value={newPosForm.accountId}
                              onChange={(e) => setNewPosForm({ ...newPosForm, accountId: e.target.value })}
                            >
                              <option value="">– wählen –</option>
                              {accounts.map((a) => (
                                <option key={a.id} value={a.id}>{a.name}</option>
                              ))}
                            </select>
                          </div>
                        )}
                      </div>
                      <div className="form-row">
                        <div className="form-group">
                          <label>Frequenz</label>
                          <select
                            value={newPosForm.frequencyType}
                            onChange={(e) => setNewPosForm({ ...newPosForm, frequencyType: e.target.value })}
                          >
                            <option value="daily">Täglich</option>
                            <option value="weekly">Wöchentlich</option>
                            <option value="biweekly">Alle 2 Wochen</option>
                            <option value="monthly">Monatlich</option>
                            <option value="quarterly">Quartalsweise</option>
                            <option value="semi_annually">Halbjährlich</option>
                            <option value="annually">Jährlich</option>
                          </select>
                        </div>
                        <div className="form-group">
                          <label>Intervall</label>
                          <input
                            type="number"
                            min="1"
                            value={newPosForm.interval}
                            onChange={(e) => setNewPosForm({ ...newPosForm, interval: e.target.value })}
                          />
                        </div>
                      </div>
                      <div className="form-row">
                        <div className="form-group">
                          <label>Tag im Monat</label>
                          <input
                            type="number"
                            min="1"
                            max="31"
                            value={newPosForm.dayOfMonth}
                            onChange={(e) => setNewPosForm({ ...newPosForm, dayOfMonth: e.target.value })}
                          />
                        </div>
                        <div className="form-group">
                          <label>Geschäftstag-Regel</label>
                          <select
                            value={newPosForm.businessDayRule}
                            onChange={(e) => setNewPosForm({ ...newPosForm, businessDayRule: e.target.value })}
                          >
                            <option value="exact">Exakt</option>
                            <option value="last_business_day_before">Letzter Geschäftstag davor</option>
                            <option value="first_business_day_after">Erster Geschäftstag danach</option>
                            <option value="last_business_day_of_month">Letzter Geschäftstag im Monat</option>
                          </select>
                        </div>
                      </div>
                      <div className="form-row">
                        <div className="form-group">
                          <label>Startdatum</label>
                          <input
                            type="date"
                            value={newPosForm.startDate}
                            onChange={(e) => setNewPosForm({ ...newPosForm, startDate: e.target.value })}
                            required
                          />
                        </div>
                        <div className="form-group">
                          <label>Enddatum (optional)</label>
                          <input
                            type="date"
                            value={newPosForm.endDate}
                            onChange={(e) => setNewPosForm({ ...newPosForm, endDate: e.target.value })}
                          />
                        </div>
                        <div className="form-group">
                          <label>Dynamik (% p.a.)</label>
                          <input
                            type="number"
                            step="0.1"
                            value={newPosForm.growthRate}
                            onChange={(e) => setNewPosForm({ ...newPosForm, growthRate: e.target.value })}
                            placeholder="z.B. 2.0"
                          />
                        </div>
                      </div>
                      <div className="form-actions">
                        <button type="button" className="btn btn-sm btn-secondary" onClick={() => setShowNewPosForm(false)}>
                          Abbrechen
                        </button>
                        <button type="submit" className="btn btn-sm btn-primary">
                          Hinzufügen
                        </button>
                      </div>
                    </form>
                  </div>
                ) : (
                  <button
                    className="btn btn-sm btn-success scenario-add-btn"
                    onClick={() => setShowNewPosForm(true)}
                  >
                    + Neue Position hinzufügen
                  </button>
                )}
              </div>
            )}
          </div>

          {/* Summary Cards */}
          <div className="dashboard-grid">
            <div className="card stat-card">
              <div className="stat-icon">💰</div>
              <div className="stat-value">{formatCurrency(totalStart)}</div>
              <div className="stat-label">Aktueller Gesamtsaldo (€)</div>
            </div>
            <div className="card stat-card">
              <div className="stat-icon">📈</div>
              <div className="stat-value">{formatCurrency(totalEnd)}</div>
              <div className="stat-label">Projizierter Saldo (€)</div>
            </div>
            <div className="card stat-card">
              <div className="stat-icon">{totalChange >= 0 ? "✅" : "⚠️"}</div>
              <div className={`stat-value ${totalChange >= 0 ? "amount-positive" : "amount-negative"}`}>
                {totalChange >= 0 ? "+" : ""}
                {formatCurrency(totalChange)}
              </div>
              <div className="stat-label">Veränderung (€)</div>
            </div>
            {data?.inflationAdjustedTotals && data.inflationAdjustedTotals.length > 0 ? (
              <div className="card stat-card">
                <div className="stat-icon">📉</div>
                <div className="stat-value">
                  {formatCurrency(data.inflationAdjustedTotals[data.inflationAdjustedTotals.length - 1]?.balance ?? 0)}
                </div>
                <div className="stat-label">Kaufkraftbereinigt (€)</div>
              </div>
            ) : (
              <div className="card stat-card">
                <div className="stat-icon">🏦</div>
                <div className="stat-value">{filteredAccounts.length}</div>
                <div className="stat-label">Konten</div>
              </div>
            )}
          </div>

          {/* Main Chart */}
          {chartData.length > 0 ? (
            <div className="card projection-chart-card">
              <h3>Kontostände im Zeitverlauf</h3>
              <ResponsiveContainer width="100%" height={400}>
                <LineChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                  <XAxis
                    dataKey="dateLabel"
                    tick={{ fontSize: 12 }}
                    interval={tickInterval}
                  />
                  <YAxis
                    tick={{ fontSize: 12 }}
                    tickFormatter={(v: number) => formatCurrency(v)}
                    width={100}
                  />
                  <Tooltip
                    formatter={(value) => [formatCurrency(Number(value)) + " €", undefined]}
                    labelFormatter={(label) => `Datum: ${String(label)}`}
                  />
                  <Legend />
                  <ReferenceLine y={0} stroke="#e53e3e" strokeDasharray="3 3" />
                  {filteredAccounts.map((acc, i) => (
                    <Line
                      key={acc.id}
                      type="monotone"
                      dataKey={acc.name}
                      stroke={COLORS[i % COLORS.length]}
                      strokeWidth={2}
                      dot={false}
                      activeDot={{ r: 4 }}
                    />
                  ))}
                  {filteredAccounts.map((acc, i) =>
                    acc.inflationAdjustedDataPoints && acc.inflationAdjustedDataPoints.length > 0 ? (
                      <Line
                        key={`adj-${acc.id}`}
                        type="monotone"
                        dataKey={`${acc.name} (real)`}
                        stroke={COLORS[i % COLORS.length]}
                        strokeWidth={1}
                        strokeDasharray="3 6"
                        dot={false}
                        activeDot={{ r: 3 }}
                      />
                    ) : null
                  )}
                  {filteredAccounts.length > 1 && (
                    <Line
                      type="monotone"
                      dataKey="Gesamt"
                      stroke="#1e3a5f"
                      strokeWidth={2}
                      strokeDasharray="5 5"
                      dot={false}
                      activeDot={{ r: 4 }}
                    />
                  )}
                  {filteredDepots.map((dp, i) => (
                    <Line
                      key={`depot-${dp.id}`}
                      type="monotone"
                      dataKey={`📊 ${dp.name}`}
                      stroke={COLORS[(filteredAccounts.length + i) % COLORS.length]}
                      strokeWidth={2}
                      strokeDasharray="8 4"
                      dot={false}
                      activeDot={{ r: 4 }}
                    />
                  ))}
                  {filteredDepots.map((dp, i) =>
                    dp.inflationAdjustedDataPoints && dp.inflationAdjustedDataPoints.length > 0 ? (
                      <Line
                        key={`depot-adj-${dp.id}`}
                        type="monotone"
                        dataKey={`📊 ${dp.name} (real)`}
                        stroke={COLORS[(filteredAccounts.length + i) % COLORS.length]}
                        strokeWidth={1}
                        strokeDasharray="3 6"
                        dot={false}
                        activeDot={{ r: 3 }}
                      />
                    ) : null
                  )}
                  {data?.inflationAdjustedTotals && data.inflationAdjustedTotals.length > 0 && (
                    <Line
                      type="monotone"
                      dataKey="Gesamt (inflationsbereinigt)"
                      stroke="#e53e3e"
                      strokeWidth={2}
                      strokeDasharray="3 6"
                      dot={false}
                      activeDot={{ r: 4 }}
                    />
                  )}
                </LineChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <div className="card">
              <div className="empty-state">
                <div className="empty-icon">📊</div>
                <p>Keine Kontodaten vorhanden. Erstellen Sie zuerst Konten und Positionen.</p>
              </div>
            </div>
          )}

          {/* Account Detail Table */}
          {summaryStats.length > 0 && (
            <div className="card projection-table-card">
              <h3>Kontenübersicht</h3>
              <div className="table-wrapper">
                <table>
                  <thead>
                    <tr>
                      <th>Konto</th>
                      <th>Aktuell</th>
                      <th>Projiziert</th>
                      {hasInflation && <th>Kaufkraftbereinigt</th>}
                      <th>Veränderung</th>
                      <th>⌀ mtl. Netto</th>
                      <th>Min</th>
                      <th>Max</th>
                    </tr>
                  </thead>
                  <tbody>
                    {summaryStats.map((s) => (
                      <tr key={s.name}>
                        <td>{s.name}</td>
                        <td className="amount">{formatCurrency(s.startBalance)} €</td>
                        <td className="amount">{formatCurrency(s.endBalance)} €</td>
                        {hasInflation && (
                          <td className="amount">{s.adjustedEndBalance !== null ? formatCurrency(s.adjustedEndBalance) + " €" : "–"}</td>
                        )}
                        <td className={`amount ${s.change >= 0 ? "amount-positive" : "amount-negative"}`}>
                          {s.change >= 0 ? "+" : ""}
                          {formatCurrency(s.change)} €
                        </td>
                        <td className={`amount ${s.monthlyNetFlow >= 0 ? "amount-positive" : "amount-negative"}`}>
                          {s.monthlyNetFlow >= 0 ? "+" : ""}
                          {formatCurrency(s.monthlyNetFlow)} €
                        </td>
                        <td className="amount">{formatCurrency(s.min)} €</td>
                        <td className="amount">{formatCurrency(s.max)} €</td>
                      </tr>
                    ))}
                    {summaryStats.length > 1 && (
                      <tr style={{ fontWeight: 700, borderTop: "2px solid var(--color-border)" }}>
                        <td>Gesamt</td>
                        <td className="amount">{formatCurrency(totalStart)} €</td>
                        <td className="amount">{formatCurrency(totalEnd)} €</td>
                        {hasInflation && (
                          <td className="amount">
                            {data?.inflationAdjustedTotals && data.inflationAdjustedTotals.length > 0
                              ? formatCurrency(data.inflationAdjustedTotals[data.inflationAdjustedTotals.length - 1]?.balance ?? 0) + " €"
                              : "–"}
                          </td>
                        )}
                        <td className={`amount ${totalChange >= 0 ? "amount-positive" : "amount-negative"}`}>
                          {totalChange >= 0 ? "+" : ""}
                          {formatCurrency(totalChange)} €
                        </td>
                        <td className={`amount ${totalMonthlyNetFlow >= 0 ? "amount-positive" : "amount-negative"}`}>
                          {totalMonthlyNetFlow >= 0 ? "+" : ""}
                          {formatCurrency(totalMonthlyNetFlow)} €
                        </td>
                        <td></td>
                        <td></td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Depot Detail Table */}
          {filteredDepots.length > 0 && (
            <div className="card projection-table-card">
              <h3>Depotübersicht</h3>
              <div className="table-wrapper">
                <table>
                  <thead>
                    <tr>
                      <th>Depot</th>
                      <th>Zinssatz</th>
                      <th>Aktuell</th>
                      <th>Projiziert</th>
                      {hasInflation && <th>Kaufkraftbereinigt</th>}
                      <th>Veränderung</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredDepots.map((dp) => {
                      const startBal = dp.dataPoints[0]?.balance ?? 0;
                      const endBal = dp.dataPoints[dp.dataPoints.length - 1]?.balance ?? 0;
                      const change = endBal - startBal;
                      const adjEndBal = dp.inflationAdjustedDataPoints && dp.inflationAdjustedDataPoints.length > 0
                        ? dp.inflationAdjustedDataPoints[dp.inflationAdjustedDataPoints.length - 1]?.balance ?? null
                        : null;
                      return (
                        <tr key={dp.id}>
                          <td>📊 {dp.name}</td>
                          <td>{dp.interestRate.toFixed(2)} % p.a.</td>
                          <td className="amount">{formatCurrency(startBal)} €</td>
                          <td className="amount">{formatCurrency(endBal)} €</td>
                          {hasInflation && (
                            <td className="amount">{adjEndBal !== null ? formatCurrency(adjEndBal) + " €" : "–"}</td>
                          )}
                          <td className={`amount ${change >= 0 ? "amount-positive" : "amount-negative"}`}>
                            {change >= 0 ? "+" : ""}
                            {formatCurrency(change)} €
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}
    </>
  );
}
