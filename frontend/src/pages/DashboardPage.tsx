import { useEffect, useState } from "react";
import { getPersons, getAccounts, getPositions } from "../api";

export default function DashboardPage() {
  const [counts, setCounts] = useState({ persons: 0, accounts: 0, positions: 0 });
  const [totalBalance, setTotalBalance] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([getPersons(), getAccounts(), getPositions()])
      .then(([persons, accounts, positions]) => {
        setCounts({
          persons: persons.length,
          accounts: accounts.length,
          positions: positions.length,
        });
        setTotalBalance(accounts.reduce((sum, a) => sum + a.balance, 0));
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="loading">Laden…</div>;

  return (
    <>
      <div className="page-header">
        <h2>Übersicht</h2>
      </div>

      <div className="dashboard-grid">
        <div className="card stat-card">
          <div className="stat-icon">👤</div>
          <div className="stat-value">{counts.persons}</div>
          <div className="stat-label">Personen</div>
        </div>
        <div className="card stat-card">
          <div className="stat-icon">🏦</div>
          <div className="stat-value">{counts.accounts}</div>
          <div className="stat-label">Konten</div>
        </div>
        <div className="card stat-card">
          <div className="stat-icon">📋</div>
          <div className="stat-value">{counts.positions}</div>
          <div className="stat-label">Positionen</div>
        </div>
        <div className="card stat-card">
          <div className="stat-icon">💰</div>
          <div className="stat-value">
            {totalBalance.toLocaleString("de-DE", { minimumFractionDigits: 2 })}
          </div>
          <div className="stat-label">Gesamtsaldo (€)</div>
        </div>
      </div>
    </>
  );
}
