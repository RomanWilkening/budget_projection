import { useEffect, useState, useRef, type FormEvent } from "react";
import type { Position, Account, PositionType, FrequencyType, BusinessDayRule, PositionSeparator } from "../types";
import { getPositions, createPosition, updatePosition, deletePosition, getAccounts, getSeparators, createSeparator, updateSeparator, deleteSeparator, reorderPositions } from "../api";

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
  "Sonntag",
  "Montag",
  "Dienstag",
  "Mittwoch",
  "Donnerstag",
  "Freitag",
  "Samstag",
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
  growthRate: string;
}

const emptyForm: FormState = {
  name: "",
  type: "expense",
  amount: "0",
  accountId: "",
  sourceAccountId: "",
  targetAccountId: "",
  frequencyType: "monthly",
  interval: "1",
  dayOfMonth: "",
  monthOfYear: "",
  dayOfWeek: "",
  businessDayRule: "exact",
  startDate: new Date().toISOString().slice(0, 10),
  endDate: "",
  growthRate: "0",
};

// Unified list item: either a position or a separator
type ListItem =
  | { kind: "position"; data: Position }
  | { kind: "separator"; data: PositionSeparator };

export default function PositionsPage() {
  const [positions, setPositions] = useState<Position[]>([]);
  const [separators, setSeparators] = useState<PositionSeparator[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Position | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [formError, setFormError] = useState("");
  const [separatorName, setSeparatorName] = useState("");
  const [editingSeparator, setEditingSeparator] = useState<PositionSeparator | null>(null);
  const [editingSeparatorName, setEditingSeparatorName] = useState("");

  // Drag & drop state
  const [dragItem, setDragItem] = useState<{ kind: "position" | "separator"; id: number } | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const dragCounter = useRef(0);

  const load = () => {
    setLoading(true);
    Promise.all([getPositions(), getAccounts(), getSeparators()])
      .then(([p, a, s]) => {
        setPositions(p);
        setAccounts(a);
        setSeparators(s);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  // Build merged & sorted list
  const items: ListItem[] = [
    ...positions.map((p): ListItem => ({ kind: "position", data: p })),
    ...separators.map((s): ListItem => ({ kind: "separator", data: s })),
  ].sort((a, b) => a.data.sortOrder - b.data.sortOrder);

  // Drag & drop handlers
  const handleDragStart = (e: React.DragEvent, kind: "position" | "separator", id: number) => {
    setDragItem({ kind, id });
    e.dataTransfer.effectAllowed = "move";
    const target = e.currentTarget as HTMLElement;
    target.classList.add("dragging");
  };

  const handleDragEnd = (e: React.DragEvent) => {
    const target = e.currentTarget as HTMLElement;
    target.classList.remove("dragging");
    setDragItem(null);
    setDragOverIndex(null);
    dragCounter.current = 0;
  };

  const handleDragEnter = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    dragCounter.current++;
    setDragOverIndex(index);
  };

  const handleDragLeave = () => {
    dragCounter.current--;
    if (dragCounter.current <= 0) {
      setDragOverIndex(null);
      dragCounter.current = 0;
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
  };

  const handleDrop = async (e: React.DragEvent, dropIndex: number) => {
    e.preventDefault();
    setDragOverIndex(null);
    dragCounter.current = 0;

    if (!dragItem) return;

    const dragIndex = items.findIndex(
      (item) =>
        (item.kind === "position" && dragItem.kind === "position" && item.data.id === dragItem.id) ||
        (item.kind === "separator" && dragItem.kind === "separator" && item.data.id === dragItem.id)
    );

    if (dragIndex === dropIndex || dragIndex === -1) return;

    const newItems = [...items];
    const [moved] = newItems.splice(dragIndex, 1);
    newItems.splice(dropIndex, 0, moved);

    // Optimistic update
    const reorderedPositions: Position[] = [];
    const reorderedSeparators: PositionSeparator[] = [];
    const reorderPayload: { type: string; id: number }[] = [];

    newItems.forEach((item, i) => {
      if (item.kind === "position") {
        reorderedPositions.push({ ...item.data as Position, sortOrder: i });
        reorderPayload.push({ type: "position", id: item.data.id });
      } else {
        reorderedSeparators.push({ ...item.data as PositionSeparator, sortOrder: i });
        reorderPayload.push({ type: "separator", id: item.data.id });
      }
    });

    setPositions(reorderedPositions);
    setSeparators(reorderedSeparators);
    setDragItem(null);

    try {
      await reorderPositions(reorderPayload);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler beim Sortieren");
      load();
    }
  };

  const openCreate = () => {
    setEditing(null);
    setForm(emptyForm);
    setFormError("");
    setShowForm(true);
  };

  const openEdit = (p: Position) => {
    setEditing(p);
    setForm({
      name: p.name,
      type: p.type,
      amount: String(p.amount),
      accountId: p.accountId ? String(p.accountId) : "",
      sourceAccountId: p.sourceAccountId ? String(p.sourceAccountId) : "",
      targetAccountId: p.targetAccountId ? String(p.targetAccountId) : "",
      frequencyType: p.frequencyType as FrequencyType,
      interval: String(p.interval),
      dayOfMonth: p.dayOfMonth != null ? String(p.dayOfMonth) : "",
      monthOfYear: p.monthOfYear != null ? String(p.monthOfYear) : "",
      dayOfWeek: p.dayOfWeek != null ? String(p.dayOfWeek) : "",
      businessDayRule: p.businessDayRule as BusinessDayRule,
      startDate: p.startDate.slice(0, 10),
      endDate: p.endDate ? p.endDate.slice(0, 10) : "",
      growthRate: String(p.growthRate ?? 0),
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
    form.frequencyType === "quarterly" || form.frequencyType === "semi_annually" || form.frequencyType === "annually";

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
      growthRate: parseFloat(form.growthRate) || 0,
    };

    if (form.type === "transfer") {
      payload.sourceAccountId = parseInt(form.sourceAccountId);
      payload.targetAccountId = parseInt(form.targetAccountId);
    } else {
      payload.accountId = parseInt(form.accountId);
      if (form.type === "expense" && form.targetAccountId) {
        payload.targetAccountId = parseInt(form.targetAccountId);
      }
      if (form.type === "income" && form.sourceAccountId) {
        payload.sourceAccountId = parseInt(form.sourceAccountId);
      }
    }

    if (needsDayOfMonth && form.dayOfMonth) payload.dayOfMonth = parseInt(form.dayOfMonth);
    if (needsMonthOfYear && form.monthOfYear) payload.monthOfYear = parseInt(form.monthOfYear);
    if (needsDayOfWeek && form.dayOfWeek) payload.dayOfWeek = parseInt(form.dayOfWeek);
    if (form.endDate) payload.endDate = form.endDate;

    try {
      if (editing) {
        await updatePosition(editing.id, payload);
      } else {
        await createPosition(payload);
      }
      setShowForm(false);
      load();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Fehler");
    }
  };

  const handleDelete = async (p: Position) => {
    if (!confirm(`Position "${p.name}" wirklich löschen?`)) return;
    try {
      await deletePosition(p.id);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler");
    }
  };

  const handleCreateSeparator = async () => {
    const name = separatorName.trim();
    if (!name) return;
    try {
      await createSeparator({ name });
      setSeparatorName("");
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler");
    }
  };

  const handleUpdateSeparator = async (sep: PositionSeparator) => {
    const name = editingSeparatorName.trim();
    if (!name) return;
    try {
      await updateSeparator(sep.id, { name });
      setEditingSeparator(null);
      setEditingSeparatorName("");
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler");
    }
  };

  const handleDeleteSeparator = async (sep: PositionSeparator) => {
    if (!confirm(`Trenner "${sep.name}" wirklich löschen?`)) return;
    try {
      await deleteSeparator(sep.id);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler");
    }
  };

  const typeBadge = (type: PositionType) => {
    const labels: Record<PositionType, string> = { income: "Einnahme", expense: "Ausgabe", transfer: "Umbuchung" };
    return <span className={`badge badge-${type}`}>{labels[type]}</span>;
  };

  const freqLabel = (ft: string) =>
    FREQUENCY_TYPES.find((f) => f.value === ft)?.label ?? ft;

  const formatAccountDisplay = (p: Position) => {
    if (p.type === "transfer") {
      return `${accountName(p.sourceAccountId)} → ${accountName(p.targetAccountId)}`;
    }
    if (p.type === "expense" && p.targetAccountId) {
      return `${accountName(p.accountId)} → ${accountName(p.targetAccountId)}`;
    }
    if (p.type === "income" && p.sourceAccountId) {
      return `${accountName(p.sourceAccountId)} → ${accountName(p.accountId)}`;
    }
    return accountName(p.accountId);
  };

  const accountName = (id?: number) => {
    if (!id) return "–";
    return accounts.find((a) => a.id === id)?.name ?? `#${id}`;
  };

  return (
    <>
      <div className="page-header">
        <h2>Positionen</h2>
        <div className="page-header-actions">
          <div className="separator-add-inline">
            <input
              type="text"
              value={separatorName}
              onChange={(e) => setSeparatorName(e.target.value)}
              placeholder="Trenner-Name"
              onKeyDown={(e) => {
                if (e.key === "Enter") handleCreateSeparator();
              }}
            />
            <button className="btn btn-secondary" onClick={handleCreateSeparator} disabled={!separatorName.trim()}>
              + Trenner
            </button>
          </div>
          <button className="btn btn-primary" onClick={openCreate}>
            + Neue Position
          </button>
        </div>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {loading ? (
        <div className="loading">Laden…</div>
      ) : items.length === 0 ? (
        <div className="card empty-state">
          <div className="empty-icon">📋</div>
          <p>Noch keine Positionen vorhanden.</p>
        </div>
      ) : (
        <div className="card table-wrapper">
          <table>
            <thead>
              <tr>
                <th style={{ width: 40 }}></th>
                <th>Name</th>
                <th>Typ</th>
                <th>Betrag</th>
                <th>Änderung p.a.</th>
                <th>Konto</th>
                <th>Frequenz</th>
                <th>Startdatum</th>
                <th style={{ width: 120 }}>Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item, index) => {
                if (item.kind === "separator") {
                  const sep = item.data as PositionSeparator;
                  return (
                    <tr
                      key={`sep-${sep.id}`}
                      className={`separator-row${dragOverIndex === index ? " drag-over" : ""}`}
                      draggable
                      onDragStart={(e) => handleDragStart(e, "separator", sep.id)}
                      onDragEnd={handleDragEnd}
                      onDragEnter={(e) => handleDragEnter(e, index)}
                      onDragLeave={handleDragLeave}
                      onDragOver={handleDragOver}
                      onDrop={(e) => handleDrop(e, index)}
                    >
                      <td className="drag-handle">⠿</td>
                      <td colSpan={6} className="separator-cell">
                        {editingSeparator?.id === sep.id ? (
                          <span className="separator-edit-inline">
                            <input
                              type="text"
                              value={editingSeparatorName}
                              onChange={(e) => setEditingSeparatorName(e.target.value)}
                              onKeyDown={(e) => {
                                if (e.key === "Enter") handleUpdateSeparator(sep);
                                if (e.key === "Escape") setEditingSeparator(null);
                              }}
                              autoFocus
                            />
                            <button className="btn btn-sm btn-primary" onClick={() => handleUpdateSeparator(sep)}>
                              ✓
                            </button>
                          </span>
                        ) : (
                          <span className="separator-label">{sep.name}</span>
                        )}
                      </td>
                      <td colSpan={2}>
                        <div className="actions-cell">
                          <button
                            className="btn btn-sm btn-secondary"
                            onClick={() => {
                              setEditingSeparator(sep);
                              setEditingSeparatorName(sep.name);
                            }}
                          >
                            ✏️
                          </button>
                          <button className="btn btn-sm btn-danger" onClick={() => handleDeleteSeparator(sep)}>
                            🗑️
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                }

                const p = item.data as Position;
                return (
                  <tr
                    key={`pos-${p.id}`}
                    className={dragOverIndex === index ? "drag-over" : ""}
                    draggable
                    onDragStart={(e) => handleDragStart(e, "position", p.id)}
                    onDragEnd={handleDragEnd}
                    onDragEnter={(e) => handleDragEnter(e, index)}
                    onDragLeave={handleDragLeave}
                    onDragOver={handleDragOver}
                    onDrop={(e) => handleDrop(e, index)}
                  >
                    <td className="drag-handle">⠿</td>
                    <td>{p.name}</td>
                    <td>{typeBadge(p.type)}</td>
                    <td>
                      <span className="amount">
                        {p.amount.toLocaleString("de-DE", { minimumFractionDigits: 2 })} €
                      </span>
                    </td>
                    <td>
                      {p.growthRate ? `${p.growthRate.toLocaleString("de-DE", { minimumFractionDigits: 1 })} %` : "–"}
                    </td>
                    <td>{formatAccountDisplay(p)}</td>
                    <td>{freqLabel(p.frequencyType)}</td>
                    <td>{new Date(p.startDate).toLocaleDateString("de-DE")}</td>
                    <td>
                      <div className="actions-cell">
                        <button className="btn btn-sm btn-secondary" onClick={() => openEdit(p)}>
                          ✏️
                        </button>
                        <button className="btn btn-sm btn-danger" onClick={() => handleDelete(p)}>
                          🗑️
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {showForm && (
        <div className="form-overlay" onClick={() => setShowForm(false)}>
          <div className="form-modal" onClick={(e) => e.stopPropagation()}>
            <h3>{editing ? "Position bearbeiten" : "Neue Position"}</h3>
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
                <div className="form-group">
                  <label>Änderung p.a. (%)</label>
                  <input
                    type="number"
                    step="0.1"
                    value={form.growthRate}
                    onChange={(e) => setField("growthRate", e.target.value)}
                    placeholder="z.B. 2.0"
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
                <>
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
                  {form.type === "expense" && (
                    <div className="form-group">
                      <label>Zielkonto (optional)</label>
                      <select
                        value={form.targetAccountId}
                        onChange={(e) => setField("targetAccountId", e.target.value)}
                      >
                        <option value="">— Kein Zielkonto —</option>
                        {accounts.map((a) => (
                          <option key={a.id} value={a.id}>
                            {a.name}
                          </option>
                        ))}
                      </select>
                    </div>
                  )}
                  {form.type === "income" && (
                    <div className="form-group">
                      <label>Quellkonto (optional)</label>
                      <select
                        value={form.sourceAccountId}
                        onChange={(e) => setField("sourceAccountId", e.target.value)}
                      >
                        <option value="">— Kein Quellkonto —</option>
                        {accounts.map((a) => (
                          <option key={a.id} value={a.id}>
                            {a.name}
                          </option>
                        ))}
                      </select>
                    </div>
                  )}
                </>
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
                      <label>Startmonat</label>
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
                  {editing ? "Speichern" : "Erstellen"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  );
}
