import { useEffect, useState } from "react";
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
import { getProjection } from "../api";
import type { ProjectionResponse } from "../types";

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

export default function ProjectionPage() {
  const [months, setMonths] = useState(6);
  const [data, setData] = useState<ProjectionResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const today = new Date().toISOString().split("T")[0];

  useEffect(() => {
    let cancelled = false;
    getProjection({ months, startDate: today })
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
  }, [months, today]);

  // Build chart data: merge all accounts into rows keyed by date
  const chartData = data
    ? data.totals.map((t, i) => {
        const row: Record<string, string | number> = {
          date: t.date,
          dateLabel: formatDateLabel(t.date),
          Gesamt: t.balance,
        };
        for (const acc of data.accounts) {
          row[acc.name] = acc.dataPoints[i]?.balance ?? 0;
        }
        return row;
      })
    : [];

  // Compute summary statistics
  const summaryStats = data
    ? data.accounts.map((acc) => {
        const balances = acc.dataPoints.map((dp) => dp.balance);
        const startBal = balances[0] ?? 0;
        const endBal = balances[balances.length - 1] ?? 0;
        return {
          name: acc.name,
          currency: acc.currency,
          startBalance: startBal,
          endBalance: endBal,
          change: endBal - startBal,
          min: Math.min(...balances),
          max: Math.max(...balances),
        };
      })
    : [];

  const totalStart = summaryStats.reduce((s, a) => s + a.startBalance, 0);
  const totalEnd = summaryStats.reduce((s, a) => s + a.endBalance, 0);
  const totalChange = totalEnd - totalStart;

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
              className={`btn btn-sm ${months === opt.value ? "btn-primary" : "btn-secondary"}`}
              onClick={() => setMonths(opt.value)}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {error && <div className="error-banner">{error}</div>}
      {loading && <div className="loading">Laden…</div>}

      {!loading && data && (
        <>
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
            <div className="card stat-card">
              <div className="stat-icon">🏦</div>
              <div className="stat-value">{data.accounts.length}</div>
              <div className="stat-label">Konten</div>
            </div>
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
                  {data.accounts.map((acc, i) => (
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
                  {data.accounts.length > 1 && (
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
                      <th>Veränderung</th>
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
                        <td className={`amount ${s.change >= 0 ? "amount-positive" : "amount-negative"}`}>
                          {s.change >= 0 ? "+" : ""}
                          {formatCurrency(s.change)} €
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
                        <td className={`amount ${totalChange >= 0 ? "amount-positive" : "amount-negative"}`}>
                          {totalChange >= 0 ? "+" : ""}
                          {formatCurrency(totalChange)} €
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
        </>
      )}
    </>
  );
}
