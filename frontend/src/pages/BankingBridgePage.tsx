import { useEffect, useState, type FormEvent } from "react";
import type { Account, RecurringAnalysis, RecurringPattern, PositionType, FrequencyType, BusinessDayRule } from "../types";
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

const POSITION_TYPES: { value: PositionType; label: string }[] = [
  { value: "income", label: "Einnahme" },
  { value: "expense", label: "Ausgabe" },
  { value: "transfer", label: "Umbuchung" },
];

const FREQUENCY_TYPES: { value: FrequencyType; label: string }[] = [
  { value: "daily", label: "Täglich" },
  { value: "weekly", label: "Wöchentlich" },
  { value: "biweekly", label: "Zweiwöchentlich" },
  { value: "monthly", label: "Monatlich" },
  { value: "quarterly", label: "Vierteljährlich" },
  { value: "semi_annually", label: "Halbjährlich" },
  { value: "annually", label: "Jährlich" },
];

const BUSINESS_DAY_RULES: { value: BusinessDayRule; label: string }[] = [
  { value: "exact", label: "Exakt" },
  { value: "last_business_day_before", label: "Letzter Werktag davor" },
  { value: "first_business_day_after", label: "Erster Werktag danach" },
  { value: "last_business_day_of_month", label: "Letzter Werktag des Monats" },
];

const WEEKDAYS = [
  "Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag",
];

const MONTHS = [
  "Januar", "Februar", "März", "April", "Mai", "Juni",
  "Juli", "August", "September", "Oktober", "November", "Dezember",
];

interface FormState {
  name: string;
  type: PositionType;
  amount: string;
  accountId: string;
  sourceAccountId: string;
  targetAccountId: string;
  frequencyType: FrequencyType;
  interval: string;
  dayOfMonth: string;
  monthOfYear: string;
  dayOfWeek: string;
  businessDayRule: BusinessDayRule;
  startDate: string;
  endDate: string;
}

export default function BankingBridgePage() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [linkedAccounts, setLinkedAccounts] = useState<Account[]>([]);
  const [selectedAccountId, setSelectedAccountId] = useState<string>("");
  const [months, setMonths] = useState(12);
  const [analysis, setAnalysis] = useState<RecurringAnalysis | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [created, setCreated] = useState<Set<string>>(new Set());

  // Modal form state
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<FormState>({
    name: "", type: "expense", amount: "0", accountId: "", sourceAccountId: "",
    targetAccountId: "", frequencyType: "monthly", interval: "1", dayOfMonth: "",
    monthOfYear: "", dayOfWeek: "", businessDayRule: "exact",
    startDate: new Date().toISOString().slice(0, 10), endDate: "",
  });
  const [formError, setFormError] = useState("");
  const [formPatternKey, setFormPatternKey] = useState<string | null>(null);

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

  const openCreateModal = (pattern: RecurringPattern) => {
    const key = pattern.name + pattern.counterpartIban;
    setFormPatternKey(key);
    setForm({
      name: pattern.name || "",
      type: pattern.isExpense ? "expense" : "income",
      amount: String(pattern.lastAmount),
      accountId: selectedAccountId,
      sourceAccountId: "",
      targetAccountId: "",
      frequencyType: (pattern.frequency as FrequencyType) || "monthly",
      interval: "1",
      dayOfMonth: pattern.dayOfMonth != null ? String(pattern.dayOfMonth) : "",
      monthOfYear: "",
      dayOfWeek: "",
      businessDayRule: "exact",
      startDate: new Date().toISOString().slice(0, 10),
      endDate: "",
    });
    setFormError("");
    setShowForm(true);
  };

  const setField = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const needsDayOfWeek = form.frequencyType === "weekly" || form.frequencyType === "biweekly";
  const needsDayOfMonth =
    form.frequencyType === "monthly" ||
    form.frequencyType === "quarterly" ||
    form.frequencyType === "semi_annually" ||
    form.frequencyType === "annually";
  const needsMonthOfYear =
    form.frequencyType === "semi_annually" || form.frequencyType === "annually";

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    const name = form.name.trim();
    if (!name) {
      setFormError("Name ist erforderlich.");
      return;
    }
    const amount = parseFloat(form.amount);
    if (isNaN(amount) || amount <= 0) {
      setFormError("Betrag muss größer als 0 sein.");
      return;
    }
    if (!form.startDate) {
      setFormError("Startdatum ist erforderlich.");
      return;
    }
    if (form.type === "transfer") {
      if (!form.sourceAccountId || !form.targetAccountId) {
        setFormError("Quell- und Zielkonto sind bei Umbuchungen erforderlich.");
        return;
      }
    } else if (!form.accountId) {
      setFormError("Konto ist erforderlich.");
      return;
    }

    const payload: Record<string, unknown> = {
      name,
      type: form.type,
      amount,
      frequencyType: form.frequencyType,
      interval: parseInt(form.interval) || 1,
      businessDayRule: form.businessDayRule,
      startDate: form.startDate,
    };

    if (form.type === "transfer") {
      payload.sourceAccountId = parseInt(form.sourceAccountId);
      payload.targetAccountId = parseInt(form.targetAccountId);
    } else {
      payload.accountId = parseInt(form.accountId);
    }

    if (needsDayOfMonth && form.dayOfMonth) payload.dayOfMonth = parseInt(form.dayOfMonth);
    if (needsMonthOfYear && form.monthOfYear) payload.monthOfYear = parseInt(form.monthOfYear);
    if (needsDayOfWeek && form.dayOfWeek) payload.dayOfWeek = parseInt(form.dayOfWeek);
    if (form.endDate) payload.endDate = form.endDate;

    try {
      await createPosition(payload);
      setShowForm(false);
      if (formPatternKey) {
        setCreated((prev) => new Set(prev).add(formPatternKey));
      }
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Fehler beim Erstellen");
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
                        <th>Betrag (letzter)</th>
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
                                {p.isExpense ? "−" : "+"}{formatCurrency(p.lastAmount)}
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
                                  onClick={() => openCreateModal(p)}
                                >
                                  ➕ Position
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

      {showForm && (
        <div className="form-overlay" onClick={() => setShowForm(false)}>
          <div className="form-modal" onClick={(e) => e.stopPropagation()}>
            <h3>Neue Position aus Banking Bridge</h3>
            <form onSubmit={handleSubmit}>
              <div className="form-group">
                <label>Name</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setField("name", e.target.value)}
                  autoFocus
                  placeholder="Positionsname"
                />
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>Typ</label>
                  <select
                    value={form.type}
                    onChange={(e) => setField("type", e.target.value as PositionType)}
                  >
                    {POSITION_TYPES.map((t) => (
                      <option key={t.value} value={t.value}>
                        {t.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="form-group">
                  <label>Betrag</label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    value={form.amount}
                    onChange={(e) => setField("amount", e.target.value)}
                  />
                </div>
              </div>

              {form.type === "transfer" ? (
                <div className="form-row">
                  <div className="form-group">
                    <label>Quellkonto</label>
                    <select
                      value={form.sourceAccountId}
                      onChange={(e) => setField("sourceAccountId", e.target.value)}
                    >
                      <option value="">— Auswählen —</option>
                      {accounts.map((a) => (
                        <option key={a.id} value={a.id}>
                          {a.name}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Zielkonto</label>
                    <select
                      value={form.targetAccountId}
                      onChange={(e) => setField("targetAccountId", e.target.value)}
                    >
                      <option value="">— Auswählen —</option>
                      {accounts.map((a) => (
                        <option key={a.id} value={a.id}>
                          {a.name}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>
              ) : (
                <div className="form-group">
                  <label>Konto</label>
                  <select
                    value={form.accountId}
                    onChange={(e) => setField("accountId", e.target.value)}
                  >
                    <option value="">— Auswählen —</option>
                    {accounts.map((a) => (
                      <option key={a.id} value={a.id}>
                        {a.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              <div className="form-row">
                <div className="form-group">
                  <label>Frequenz</label>
                  <select
                    value={form.frequencyType}
                    onChange={(e) => setField("frequencyType", e.target.value as FrequencyType)}
                  >
                    {FREQUENCY_TYPES.map((f) => (
                      <option key={f.value} value={f.value}>
                        {f.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="form-group">
                  <label>Intervall</label>
                  <input
                    type="number"
                    min="1"
                    value={form.interval}
                    onChange={(e) => setField("interval", e.target.value)}
                  />
                </div>
              </div>

              {needsDayOfWeek && (
                <div className="form-group">
                  <label>Wochentag</label>
                  <select
                    value={form.dayOfWeek}
                    onChange={(e) => setField("dayOfWeek", e.target.value)}
                  >
                    <option value="">— Auswählen —</option>
                    {WEEKDAYS.map((d, i) => (
                      <option key={i} value={i}>
                        {d}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              {needsDayOfMonth && (
                <div className="form-row">
                  <div className="form-group">
                    <label>Tag des Monats</label>
                    <input
                      type="number"
                      min="1"
                      max="31"
                      value={form.dayOfMonth}
                      onChange={(e) => setField("dayOfMonth", e.target.value)}
                      placeholder="1–31"
                    />
                  </div>
                  {needsMonthOfYear && (
                    <div className="form-group">
                      <label>Monat</label>
                      <select
                        value={form.monthOfYear}
                        onChange={(e) => setField("monthOfYear", e.target.value)}
                      >
                        <option value="">— Auswählen —</option>
                        {MONTHS.map((m, i) => (
                          <option key={i + 1} value={i + 1}>
                            {m}
                          </option>
                        ))}
                      </select>
                    </div>
                  )}
                </div>
              )}

              <div className="form-group">
                <label>Geschäftstag-Regel</label>
                <select
                  value={form.businessDayRule}
                  onChange={(e) => setField("businessDayRule", e.target.value as BusinessDayRule)}
                >
                  {BUSINESS_DAY_RULES.map((r) => (
                    <option key={r.value} value={r.value}>
                      {r.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>Startdatum</label>
                  <input
                    type="date"
                    value={form.startDate}
                    onChange={(e) => setField("startDate", e.target.value)}
                  />
                </div>
                <div className="form-group">
                  <label>Enddatum (optional)</label>
                  <input
                    type="date"
                    value={form.endDate}
                    onChange={(e) => setField("endDate", e.target.value)}
                  />
                </div>
              </div>

              {formError && <div className="form-error">{formError}</div>}

              <div className="form-actions">
                <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>
                  Abbrechen
                </button>
                <button type="submit" className="btn btn-primary">
                  Erstellen
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  );
}
