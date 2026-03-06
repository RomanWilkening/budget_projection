import { useEffect, useState } from "react";
import type { Account, RecurringAnalysis, RecurringPattern } from "../types";
import {
  getAccounts,
  analyzeRecurringTransactions,
  createPosition,
} from "../api";

const FREQUENCY_LABELS: Record<string, string> = {
  daily: "Täglich",
  weekly: "Wöchentlich",
  biweekly: "Zweiwöchentlich",
  monthly: "Monatlich",
  quarterly: "Vierteljährlich",
  semi_annually: "Halbjährlich",
  annually: "Jährlich",
};

export default function BankingBridgePage() {
  const [, setAccounts] = useState<Account[]>([]);
  const [linkedAccounts, setLinkedAccounts] = useState<Account[]>([]);
  const [selectedAccountId, setSelectedAccountId] = useState<string>("");
  const [months, setMonths] = useState(12);
  const [analysis, setAnalysis] = useState<RecurringAnalysis | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState<string | null>(null);
  const [created, setCreated] = useState<Set<string>>(new Set());

  useEffect(() => {
    getAccounts()
      .then((accs) => {
        setAccounts(accs);
        const linked = accs.filter((a) => a.bankingBridgeAccountId);
        setLinkedAccounts(linked);
        if (linked.length > 0) {
          setSelectedAccountId(String(linked[0].id));
        }
      })
      .catch((e) => setError(e.message));
  }, []);

  const handleAnalyze = async () => {
    if (!selectedAccountId) return;
    setLoading(true);
    setError("");
    setAnalysis(null);
    setCreated(new Set());
    try {
      const result = await analyzeRecurringTransactions(parseInt(selectedAccountId), months);
      setAnalysis(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Analyse-Fehler");
    } finally {
      setLoading(false);
    }
  };

  const handleCreatePosition = async (pattern: RecurringPattern) => {
    const key = pattern.name + pattern.counterpartIban;
    setCreating(key);
    try {
      const today = new Date().toISOString().slice(0, 10);
      await createPosition({
        name: pattern.name,
        type: pattern.isExpense ? "expense" : "income",
        amount: pattern.medianAmount,
        accountId: parseInt(selectedAccountId),
        frequencyType: pattern.frequency as "monthly" | "weekly" | "quarterly" | "annually",
        interval: 1,
        dayOfMonth: pattern.dayOfMonth ?? undefined,
        businessDayRule: "exact",
        startDate: today,
      });
      setCreated((prev) => new Set(prev).add(key));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler beim Erstellen");
    } finally {
      setCreating(null);
    }
  };

  const formatCurrency = (amount: number) =>
    amount.toLocaleString("de-DE", { style: "currency", currency: "EUR" });

  const confidenceBadge = (confidence: number) => {
    if (confidence >= 0.8) return <span className="badge badge-income">Hoch</span>;
    if (confidence >= 0.5) return <span className="badge badge-transfer">Mittel</span>;
    return <span className="badge badge-expense">Niedrig</span>;
  };

  const actionBadge = (action: string) => {
    switch (action) {
      case "create":
        return <span className="badge badge-income">Neu</span>;
      case "update":
        return <span className="badge badge-transfer">Aktualisieren</span>;
      case "none":
        return <span className="badge" style={{ background: "#e8e8e8" }}>Vorhanden</span>;
      default:
        return null;
    }
  };

  return (
    <>
      <div className="page-header">
        <h2>Banking Bridge – Transaktionsanalyse</h2>
      </div>

      {linkedAccounts.length === 0 ? (
        <div className="card empty-state">
          <div className="empty-icon">🔗</div>
          <p>Keine verknüpften Konten vorhanden.</p>
          <p style={{ fontSize: "0.9rem", color: "#666" }}>
            Verknüpfen Sie zunächst ein Konto auf der Konten-Seite mit der Banking Bridge.
          </p>
        </div>
      ) : (
        <>
          <div className="card" style={{ padding: "1rem", marginBottom: "1rem" }}>
            <div className="form-row" style={{ alignItems: "flex-end" }}>
              <div className="form-group" style={{ flex: 2 }}>
                <label>Konto</label>
                <select
                  value={selectedAccountId}
                  onChange={(e) => setSelectedAccountId(e.target.value)}
                >
                  {linkedAccounts.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.name} ({formatCurrency(a.balance)})
                    </option>
                  ))}
                </select>
              </div>
              <div className="form-group" style={{ flex: 1 }}>
                <label>Analysezeitraum (Monate)</label>
                <input
                  type="number"
                  min="1"
                  max="24"
                  value={months}
                  onChange={(e) => setMonths(parseInt(e.target.value) || 12)}
                />
              </div>
              <div className="form-group">
                <button
                  className="btn btn-primary"
                  onClick={handleAnalyze}
                  disabled={loading || !selectedAccountId}
                >
                  {loading ? "⏳ Analysiere…" : "🔍 Analysieren"}
                </button>
              </div>
            </div>
          </div>

          {error && <div className="error-banner">{error}</div>}

          {analysis && (
            <>
              <div className="card" style={{ padding: "1rem", marginBottom: "1rem" }}>
                <div style={{ display: "flex", gap: "2rem", fontSize: "0.9rem" }}>
                  <span>📊 <strong>{analysis.transactionCount}</strong> Transaktionen analysiert</span>
                  <span>📅 {analysis.analyzedFrom} bis {analysis.analyzedTo}</span>
                  <span>🔄 <strong>{analysis.patterns?.length ?? 0}</strong> wiederkehrende Muster erkannt</span>
                </div>
              </div>

              {(!analysis.patterns || analysis.patterns.length === 0) ? (
                <div className="card empty-state">
                  <div className="empty-icon">🔍</div>
                  <p>Keine wiederkehrenden Transaktionen erkannt.</p>
                </div>
              ) : (
                <div className="card table-wrapper">
                  <table>
                    <thead>
                      <tr>
                        <th>Name</th>
                        <th>Typ</th>
                        <th>Betrag (Ø)</th>
                        <th>Bereich</th>
                        <th>Frequenz</th>
                        <th>Vorkommen</th>
                        <th>Konfidenz</th>
                        <th>Status</th>
                        <th style={{ width: 120 }}>Aktion</th>
                      </tr>
                    </thead>
                    <tbody>
                      {analysis.patterns.map((p) => {
                        const key = p.name + p.counterpartIban;
                        const isCreated = created.has(key);
                        return (
                          <tr key={key}>
                            <td>
                              <div>
                                <strong>{p.name}</strong>
                                {p.description && (
                                  <div style={{ fontSize: "0.8rem", color: "#666" }}>{p.description}</div>
                                )}
                              </div>
                            </td>
                            <td>
                              <span className={`badge badge-${p.isExpense ? "expense" : "income"}`}>
                                {p.isExpense ? "Ausgabe" : "Einnahme"}
                              </span>
                            </td>
                            <td>
                              <span className={`amount ${p.isExpense ? "amount-negative" : "amount-positive"}`}>
                                {p.isExpense ? "−" : "+"}{formatCurrency(p.averageAmount)}
                              </span>
                            </td>
                            <td style={{ fontSize: "0.85rem" }}>
                              {formatCurrency(p.minAmount)} – {formatCurrency(p.maxAmount)}
                            </td>
                            <td>
                              {FREQUENCY_LABELS[p.frequency] ?? p.frequency}
                              {p.dayOfMonth && ` (${p.dayOfMonth}.)`}
                            </td>
                            <td>{p.occurrences}×</td>
                            <td>{confidenceBadge(p.confidence)}</td>
                            <td>
                              {actionBadge(p.suggestedAction)}
                              {p.matchingPositionName && (
                                <div style={{ fontSize: "0.8rem", color: "#666" }}>
                                  → {p.matchingPositionName}
                                </div>
                              )}
                            </td>
                            <td>
                              {p.suggestedAction !== "none" && !isCreated && (
                                <button
                                  className="btn btn-sm btn-primary"
                                  onClick={() => handleCreatePosition(p)}
                                  disabled={creating === key}
                                >
                                  {creating === key ? "⏳" : "➕ Position"}
                                </button>
                              )}
                              {isCreated && (
                                <span style={{ color: "#2e7d32", fontSize: "0.85rem" }}>✅ Erstellt</span>
                              )}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </>
      )}
    </>
  );
}
