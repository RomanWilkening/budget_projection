import { useEffect, useState, type FormEvent } from "react";
import type { Account, Person } from "../types";
import {
  getAccounts,
  createAccount,
  updateAccount,
  deleteAccount,
  getPersons,
  addAccountOwner,
  removeAccountOwner,
} from "../api";

interface FormState {
  name: string;
  balance: string;
  currency: string;
  ownerIds: number[];
}

const emptyForm: FormState = { name: "", balance: "0", currency: "EUR", ownerIds: [] };

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [persons, setPersons] = useState<Person[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Account | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [formError, setFormError] = useState("");

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

  useEffect(load, []);

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
      ownerIds: a.owners?.map((o) => o.id) ?? [],
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
        await updateAccount(editing.id, { name, balance, currency });
        const existingIds = editing.owners?.map((o) => o.id) ?? [];
        const toAdd = form.ownerIds.filter((id) => !existingIds.includes(id));
        const toRemove = existingIds.filter((id) => !form.ownerIds.includes(id));
        await Promise.all([
          ...toAdd.map((pid) => addAccountOwner(editing.id, pid)),
          ...toRemove.map((pid) => removeAccountOwner(editing.id, pid)),
        ]);
      } else {
        const created = await createAccount({ name, balance, currency });
        await Promise.all(form.ownerIds.map((pid) => addAccountOwner(created.id, pid)));
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

  const formatCurrency = (amount: number, currency: string) =>
    amount.toLocaleString("de-DE", { style: "currency", currency });

  return (
    <>
      <div className="page-header">
        <h2>Konten</h2>
        <button className="btn btn-primary" onClick={openCreate}>
          + Neues Konto
        </button>
      </div>

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
                <th>Name</th>
                <th>Kontostand</th>
                <th>Währung</th>
                <th>Inhaber</th>
                <th style={{ width: 120 }}>Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {accounts.map((a) => (
                <tr key={a.id}>
                  <td>{a.name}</td>
                  <td>
                    <span className={`amount ${a.balance >= 0 ? "amount-positive" : "amount-negative"}`}>
                      {formatCurrency(a.balance, a.currency)}
                    </span>
                  </td>
                  <td>{a.currency}</td>
                  <td>
                    <div className="tag-list">
                      {a.owners?.map((o) => (
                        <span key={o.id} className="tag">
                          {o.name}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td>
                    <div className="actions-cell">
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
