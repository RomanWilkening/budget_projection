import { useEffect, useState, useRef, type FormEvent } from "react";
import type { Account, Person, BridgeAccount, BridgeStatus } from "../types";
import {
  getAccounts,
  createAccount,
  updateAccount,
  deleteAccount,
  getPersons,
  addAccountOwner,
  removeAccountOwner,
  getBankingBridgeStatus,
  getBankingBridgeAccounts,
  linkBankingBridgeAccount,
  syncAccountBalance,
  syncAllBalances,
  reorderAccounts,
} from "../api";

interface FormState {
  name: string;
  balance: string;
  currency: string;
  showInProjection: boolean;
  ownerIds: number[];
  bankingBridgeAccountId: string;
}

const emptyForm: FormState = { name: "", balance: "0", currency: "EUR", showInProjection: true, ownerIds: [], bankingBridgeAccountId: "" };

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [persons, setPersons] = useState<Person[]>([]);
  const [bridgeStatus, setBridgeStatus] = useState<BridgeStatus | null>(null);
  const [bridgeAccounts, setBridgeAccounts] = useState<BridgeAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Account | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [formError, setFormError] = useState("");
  const [syncing, setSyncing] = useState<number | null>(null);
  const [syncingAll, setSyncingAll] = useState(false);

  // Drag & drop state
  const [dragItem, setDragItem] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const dragCounter = useRef(0);

  const load = () => {
    setLoading(true);
    Promise.all([getAccounts(), getPersons()])
      .then(([a, p]) => {
        setAccounts(a);
        setPersons(p);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  };

  const loadBridgeStatus = () => {
    getBankingBridgeStatus()
      .then((s) => {
        setBridgeStatus(s);
        if (s.connected) {
          getBankingBridgeAccounts()
            .then(setBridgeAccounts)
            .catch(() => setBridgeAccounts([]));
        }
      })
      .catch(() => setBridgeStatus(null));
  };

  useEffect(() => {
    load();
    loadBridgeStatus();
  }, []);

  const openCreate = () => {
    setEditing(null);
    setForm(emptyForm);
    setFormError("");
    setShowForm(true);
  };

  const openEdit = (a: Account) => {
    setEditing(a);
    setForm({
      name: a.name,
      balance: String(a.balance),
      currency: a.currency,
      showInProjection: a.showInProjection,
      ownerIds: a.owners?.map((o) => o.id) ?? [],
      bankingBridgeAccountId: a.bankingBridgeAccountId ? String(a.bankingBridgeAccountId) : "",
    });
    setFormError("");
    setShowForm(true);
  };

  const toggleOwner = (id: number) => {
    setForm((prev) => ({
      ...prev,
      ownerIds: prev.ownerIds.includes(id)
        ? prev.ownerIds.filter((x) => x !== id)
        : [...prev.ownerIds, id],
    }));
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    const name = form.name.trim();
    if (!name) {
      setFormError("Name ist erforderlich.");
      return;
    }
    const balance = parseFloat(form.balance);
    if (isNaN(balance)) {
      setFormError("Kontostand muss eine Zahl sein.");
      return;
    }
    const currency = form.currency.trim() || "EUR";

    try {
      if (editing) {
        await updateAccount(editing.id, { name, balance, currency, showInProjection: form.showInProjection });
        const existingIds = editing.owners?.map((o) => o.id) ?? [];
        const toAdd = form.ownerIds.filter((id) => !existingIds.includes(id));
        const toRemove = existingIds.filter((id) => !form.ownerIds.includes(id));
        await Promise.all([
          ...toAdd.map((pid) => addAccountOwner(editing.id, pid)),
          ...toRemove.map((pid) => removeAccountOwner(editing.id, pid)),
        ]);
        // Update Banking Bridge linking
        const newBridgeId = form.bankingBridgeAccountId ? parseInt(form.bankingBridgeAccountId) : null;
        const oldBridgeId = editing.bankingBridgeAccountId ?? null;
        if (newBridgeId !== oldBridgeId) {
          await linkBankingBridgeAccount(editing.id, newBridgeId);
        }
      } else {
        const created = await createAccount({ name, balance, currency, showInProjection: form.showInProjection });
        await Promise.all(form.ownerIds.map((pid) => addAccountOwner(created.id, pid)));
        // Link to Banking Bridge if selected
        if (form.bankingBridgeAccountId) {
          await linkBankingBridgeAccount(created.id, parseInt(form.bankingBridgeAccountId));
        }
      }
      setShowForm(false);
      load();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Fehler");
    }
  };

  const handleDelete = async (a: Account) => {
    if (!confirm(`Konto "${a.name}" wirklich löschen?`)) return;
    try {
      await deleteAccount(a.id);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler");
    }
  };

  const handleSyncBalance = async (a: Account) => {
    setSyncing(a.id);
    try {
      await syncAccountBalance(a.id);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Sync-Fehler");
    } finally {
      setSyncing(null);
    }
  };

  const handleSyncAll = async () => {
    setSyncingAll(true);
    try {
      await syncAllBalances();
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Sync-Fehler");
    } finally {
      setSyncingAll(false);
    }
  };

  const getBridgeAccountName = (bridgeId: number | null | undefined) => {
    if (!bridgeId) return null;
    const ba = bridgeAccounts.find((b) => b.id === bridgeId);
    return ba ? `${ba.name} (${ba.bank})` : `#${bridgeId}`;
  };

  // Drag & drop handlers
  const handleDragStart = (e: React.DragEvent, id: number) => {
    setDragItem(id);
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

    if (dragItem === null) return;

    const dragIndex = accounts.findIndex((a) => a.id === dragItem);
    if (dragIndex === dropIndex || dragIndex === -1) return;

    const newItems = [...accounts];
    const [moved] = newItems.splice(dragIndex, 1);
    newItems.splice(dropIndex, 0, moved);

    setAccounts(newItems);
    setDragItem(null);

    try {
      await reorderAccounts(newItems.map((a) => a.id));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler beim Sortieren");
      load();
    }
  };

  const hasLinkedAccounts = accounts.some((a) => a.bankingBridgeAccountId);

  const formatCurrency = (amount: number, currency: string) =>
    amount.toLocaleString("de-DE", { style: "currency", currency });

  return (
    <>
      <div className="page-header">
        <h2>Konten</h2>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          {hasLinkedAccounts && (
            <button
              className="btn btn-secondary"
              onClick={handleSyncAll}
              disabled={syncingAll}
            >
              {syncingAll ? "⏳ Synchronisiere…" : "🔄 Alle Salden aktualisieren"}
            </button>
          )}
          <button className="btn btn-primary" onClick={openCreate}>
            + Neues Konto
          </button>
        </div>
      </div>

      {bridgeStatus && (
        <div className={`card ${bridgeStatus.connected ? "" : "empty-state"}`} style={{ padding: "0.75rem 1rem", marginBottom: "1rem" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span>{bridgeStatus.connected ? "🟢" : bridgeStatus.configured ? "🔴" : "⚪"}</span>
            <span style={{ fontSize: "0.9rem" }}>
              Banking Bridge: {bridgeStatus.connected ? "Verbunden" : bridgeStatus.configured ? "Nicht erreichbar" : "Nicht konfiguriert"}
              {bridgeStatus.connected && bridgeAccounts.length > 0 && ` (${bridgeAccounts.length} Konten verfügbar)`}
            </span>
          </div>
        </div>
      )}

      {error && <div className="error-banner">{error}</div>}

      {loading ? (
        <div className="loading">Laden…</div>
      ) : accounts.length === 0 ? (
        <div className="card empty-state">
          <div className="empty-icon">🏦</div>
          <p>Noch keine Konten vorhanden.</p>
        </div>
      ) : (
        <div className="card table-wrapper">
          <table>
            <thead>
              <tr>
                <th style={{ width: 40 }}></th>
                <th>Name</th>
                <th>Kontostand</th>
                <th>Währung</th>
                <th>Projektion</th>
                <th>Inhaber</th>
                {bridgeStatus?.connected && <th>Banking Bridge</th>}
                <th style={{ width: 160 }}>Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {accounts.map((a, index) => (
                <tr
                  key={a.id}
                  className={dragOverIndex === index ? "drag-over" : ""}
                  draggable
                  onDragStart={(e) => handleDragStart(e, a.id)}
                  onDragEnd={handleDragEnd}
                  onDragEnter={(e) => handleDragEnter(e, index)}
                  onDragLeave={handleDragLeave}
                  onDragOver={handleDragOver}
                  onDrop={(e) => handleDrop(e, index)}
                >
                  <td className="drag-handle">⠿</td>
                  <td>{a.name}</td>
                  <td>
                    <span className={`amount ${a.balance >= 0 ? "amount-positive" : "amount-negative"}`}>
                      {formatCurrency(a.balance, a.currency)}
                    </span>
                  </td>
                  <td>{a.currency}</td>
                  <td>
                    <span style={{ color: a.showInProjection ? "#2e7d32" : "#999", fontSize: "0.9rem" }}>
                      {a.showInProjection ? "✅ Ja" : "— Nein"}
                    </span>
                  </td>
                  <td>
                    <div className="tag-list">
                      {a.owners?.map((o) => (
                        <span key={o.id} className="tag">
                          {o.name}
                        </span>
                      ))}
                    </div>
                  </td>
                  {bridgeStatus?.connected && (
                    <td>
                      {a.bankingBridgeAccountId ? (
                        <span className="tag" style={{ background: "#e8f5e9", color: "#2e7d32" }}>
                          🔗 {getBridgeAccountName(a.bankingBridgeAccountId)}
                        </span>
                      ) : (
                        <span style={{ color: "#999", fontSize: "0.85rem" }}>Nicht verknüpft</span>
                      )}
                    </td>
                  )}
                  <td>
                    <div className="actions-cell">
                      {a.bankingBridgeAccountId && (
                        <button
                          className="btn btn-sm btn-secondary"
                          onClick={() => handleSyncBalance(a)}
                          disabled={syncing === a.id}
                          title="Kontostand aktualisieren"
                        >
                          {syncing === a.id ? "⏳" : "🔄"}
                        </button>
                      )}
                      <button className="btn btn-sm btn-secondary" onClick={() => openEdit(a)}>
                        ✏️
                      </button>
                      <button className="btn btn-sm btn-danger" onClick={() => handleDelete(a)}>
                        🗑️
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showForm && (
        <div className="form-overlay" onClick={() => setShowForm(false)}>
          <div className="form-modal" onClick={(e) => e.stopPropagation()}>
            <h3>{editing ? "Konto bearbeiten" : "Neues Konto"}</h3>
            <form onSubmit={handleSubmit}>
              <div className="form-group">
                <label>Name</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  autoFocus
                  placeholder="Kontoname"
                />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Kontostand</label>
                  <input
                    type="number"
                    step="0.01"
                    value={form.balance}
                    onChange={(e) => setForm({ ...form, balance: e.target.value })}
                  />
                </div>
                <div className="form-group">
                  <label>Währung</label>
                  <select
                    value={form.currency}
                    onChange={(e) => setForm({ ...form, currency: e.target.value })}
                  >
                    <option value="EUR">EUR</option>
                    <option value="USD">USD</option>
                    <option value="GBP">GBP</option>
                    <option value="CHF">CHF</option>
                  </select>
                </div>
              </div>
              <div className="form-group">
                <label>
                  <input
                    type="checkbox"
                    checked={form.showInProjection}
                    onChange={(e) => setForm({ ...form, showInProjection: e.target.checked })}
                  />{" "}
                  In Projektion anzeigen
                </label>
              </div>
              {persons.length > 0 && (
                <div className="form-group">
                  <label>Inhaber</label>
                  <div className="multi-select">
                    {persons.map((p) => (
                      <label key={p.id}>
                        <input
                          type="checkbox"
                          checked={form.ownerIds.includes(p.id)}
                          onChange={() => toggleOwner(p.id)}
                        />
                        {p.name}
                      </label>
                    ))}
                  </div>
                </div>
              )}
              {bridgeStatus?.connected && bridgeAccounts.length > 0 && (
                <div className="form-group">
                  <label>Banking Bridge Konto</label>
                  <select
                    value={form.bankingBridgeAccountId}
                    onChange={(e) => setForm({ ...form, bankingBridgeAccountId: e.target.value })}
                  >
                    <option value="">— Nicht verknüpft —</option>
                    {bridgeAccounts.map((ba) => (
                      <option key={ba.id} value={ba.id}>
                        {ba.name} ({ba.bank}) – {ba.iban}
                      </option>
                    ))}
                  </select>
                </div>
              )}
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
