import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Legend,
} from "recharts";
import { getPersons, getAccounts, getPositions } from "../api";
import type { Person, Account, Position } from "../types";

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

function formatCurrency(value: number): string {
  return value.toLocaleString("de-DE", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

export default function DashboardPage() {
  const [persons, setPersons] = useState<Person[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [positions, setPositions] = useState<Position[]>([]);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    Promise.all([getPersons(), getAccounts(), getPositions()])
      .then(([p, a, pos]) => {
        setPersons(p);
        setAccounts(a);
        setPositions(pos);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="loading">Laden…</div>;

  const totalBalance = accounts.reduce((sum, a) => sum + a.balance, 0);

  // Income vs Expense summary from positions
  const monthlyIncome = positions
    .filter((p) => p.type === "income")
    .reduce((sum, p) => sum + estimateMonthlyAmount(p), 0);
  const monthlyExpense = positions
    .filter((p) => p.type === "expense")
    .reduce((sum, p) => sum + estimateMonthlyAmount(p), 0);
  const monthlyNet = monthlyIncome - monthlyExpense;

  // Account balances for bar chart
  const accountBarData = accounts.map((a) => ({
    name: a.name,
    Saldo: a.balance,
  }));

  // Position type breakdown for pie chart
  const incomeCount = positions.filter((p) => p.type === "income").length;
  const expenseCount = positions.filter((p) => p.type === "expense").length;
  const transferCount = positions.filter((p) => p.type === "transfer").length;
  const pieData = [
    { name: "Einnahmen", value: incomeCount, color: "#38a169" },
    { name: "Ausgaben", value: expenseCount, color: "#e53e3e" },
    { name: "Umbuchungen", value: transferCount, color: "#3182ce" },
  ].filter((d) => d.value > 0);

  // Per-person account summary
  const personSummary = persons.map((p) => {
    const ownedAccounts = accounts.filter(
      (a) => a.owners && a.owners.some((o) => o.id === p.id)
    );
    const total = ownedAccounts.reduce((sum, a) => sum + a.balance, 0);
    return {
      name: p.name,
      accountCount: ownedAccounts.length,
      totalBalance: total,
    };
  });

  return (
    <>
      <div className="page-header">
        <h2>Übersicht</h2>
      </div>

      {/* Summary stat cards */}
      <div className="dashboard-grid">
        <div className="card stat-card">
          <div className="stat-icon">👤</div>
          <div className="stat-value">{persons.length}</div>
          <div className="stat-label">Personen</div>
        </div>
        <div className="card stat-card">
          <div className="stat-icon">🏦</div>
          <div className="stat-value">{accounts.length}</div>
          <div className="stat-label">Konten</div>
        </div>
        <div className="card stat-card">
          <div className="stat-icon">📋</div>
          <div className="stat-value">{positions.length}</div>
          <div className="stat-label">Positionen</div>
        </div>
        <div className="card stat-card">
          <div className="stat-icon">💰</div>
          <div className="stat-value">
            {formatCurrency(totalBalance)}
          </div>
          <div className="stat-label">Gesamtsaldo (€)</div>
        </div>
      </div>

      {/* Monthly estimate cards */}
      {positions.length > 0 && (
        <div className="dashboard-grid">
          <div className="card stat-card">
            <div className="stat-icon">📥</div>
            <div className="stat-value amount-positive">
              +{formatCurrency(monthlyIncome)}
            </div>
            <div className="stat-label">Monatl. Einnahmen (€)</div>
          </div>
          <div className="card stat-card">
            <div className="stat-icon">📤</div>
            <div className="stat-value amount-negative">
              -{formatCurrency(monthlyExpense)}
            </div>
            <div className="stat-label">Monatl. Ausgaben (€)</div>
          </div>
          <div className="card stat-card">
            <div className="stat-icon">{monthlyNet >= 0 ? "✅" : "⚠️"}</div>
            <div className={`stat-value ${monthlyNet >= 0 ? "amount-positive" : "amount-negative"}`}>
              {monthlyNet >= 0 ? "+" : ""}{formatCurrency(monthlyNet)}
            </div>
            <div className="stat-label">Monatl. Netto (€)</div>
          </div>
          <div className="card stat-card clickable" onClick={() => navigate("/projektion")}>
            <div className="stat-icon">📈</div>
            <div className="stat-value" style={{ fontSize: "1.2rem" }}>Zur Projektion →</div>
            <div className="stat-label">Detaillierte Vorschau</div>
          </div>
        </div>
      )}

      {/* Charts row */}
      <div className="dashboard-charts-row">
        {/* Account balances bar chart */}
        {accounts.length > 0 && (
          <div className="card dashboard-chart-card">
            <h3>Kontostände</h3>
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={accountBarData} margin={{ top: 10, right: 20, left: 10, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                <YAxis tick={{ fontSize: 12 }} tickFormatter={(v: number) => formatCurrency(v)} width={90} />
                <Tooltip formatter={(value) => [formatCurrency(Number(value)) + " €", "Saldo"]} />
                <Bar dataKey="Saldo" radius={[4, 4, 0, 0]}>
                  {accountBarData.map((_entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}

        {/* Position type pie chart */}
        {pieData.length > 0 && (
          <div className="card dashboard-chart-card">
            <h3>Positionen nach Typ</h3>
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie
                  data={pieData}
                  cx="50%"
                  cy="50%"
                  outerRadius={80}
                  dataKey="value"
                  label={({ name, value }) => `${name ?? ""}: ${value ?? ""}`}
                >
                  {pieData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Legend />
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      {/* Per-person summary table */}
      {personSummary.length > 0 && (
        <div className="card dashboard-table-card">
          <h3>Vermögen pro Person</h3>
          <div className="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>Person</th>
                  <th>Konten</th>
                  <th>Gesamtsaldo</th>
                </tr>
              </thead>
              <tbody>
                {personSummary.map((p) => (
                  <tr key={p.name}>
                    <td>{p.name}</td>
                    <td>{p.accountCount}</td>
                    <td className="amount">
                      {formatCurrency(p.totalBalance)} €
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </>
  );
}

/**
 * Estimate monthly amount from a position based on its frequency.
 */
function estimateMonthlyAmount(p: Position): number {
  const interval = p.interval || 1;
  switch (p.frequencyType) {
    case "daily":
      return (p.amount / interval) * 30;
    case "weekly":
      return (p.amount / interval) * (52 / 12);
    case "biweekly":
      return (p.amount / interval) * (26 / 12);
    case "monthly":
      return p.amount / interval;
    case "quarterly":
      return p.amount / (3 * interval);
    case "semi_annually":
      return p.amount / (6 * interval);
    case "annually":
      return p.amount / (12 * interval);
    default:
      return p.amount;
  }
}
