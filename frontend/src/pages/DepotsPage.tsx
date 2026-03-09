import { useEffect, useState, useRef, type FormEvent } from "react";
import type { Depot, Account } from "../types";
import {
  getDepots,
  createDepot,
  updateDepot,
  deleteDepot,
  getAccounts,
  reorderDepots,
} from "../api";

interface FormState {
  name: string;
  interestRate: string;
  accountIds: number[];
}

const emptyForm: FormState = { name: "", interestRate: "0", accountIds: [] };

function formatCurrency(value: number): string {
  return value.toLocaleString("de-DE", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

export default function DepotsPage() {
  const [depots, setDepots] = useState<Depot[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Depot | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [formError, setFormError] = useState("");

  // Drag & drop state
  const [dragItem, setDragItem] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const dragCounter = useRef(0);

  const load = () => {
    setLoading(true);
    Promise.all([getDepots(), getAccounts()])
      .then(([d, a]) => {
        setDepots(d);
        setAccounts(a);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  const openCreate = () => {
    setEditing(null);
    setForm(emptyForm);
    setFormError("");
    setShowForm(true);
  };

  const openEdit = (d: Depot) => {
    setEditing(d);
    setForm({
      name: d.name,
      interestRate: String(d.interestRate),
      accountIds: d.accounts?.map((a) => a.id) ?? [],
    });
    setFormError("");
    setShowForm(true);
  };

  const toggleAccount = (id: number) => {
    setForm((prev) => ({
      ...prev,
      accountIds: prev.accountIds.includes(id)
        ? prev.accountIds.filter((x) => x !== id)
        : [...prev.accountIds, id],
    }));
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    const name = form.name.trim();
    if (!name) {
      setFormError("Name ist erforderlich.");
      return;
    }
    const interestRate = parseFloat(form.interestRate);
    if (isNaN(interestRate)) {
      setFormError("Zinssatz muss eine Zahl sein.");
      return;
    }

    try {
      if (editing) {
        await updateDepot(editing.id, {
          name,
          interestRate,
          accountIds: form.accountIds,
        });
      } else {
        await createDepot({
          name,
          interestRate,
          accountIds: form.accountIds,
        });
      }
      setShowForm(false);
      load();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Fehler");
    }
  };

  const handleDelete = async (d: Depot) => {
    if (!confirm(`Depot "${d.name}" wirklich löschen?`)) return;
    try {
      await deleteDepot(d.id);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler");
    }
  };

  const getDepotBalance = (d: Depot): number => {
    if (!d.accounts) return 0;
    return d.accounts.reduce((sum, a) => sum + a.balance, 0);
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

    const dragIndex = depots.findIndex((d) => d.id === dragItem);
    if (dragIndex === dropIndex || dragIndex === -1) return;

    const newItems = [...depots];
    const [moved] = newItems.splice(dragIndex, 1);
    newItems.splice(dropIndex, 0, moved);

    setDepots(newItems);
    setDragItem(null);

    try {
      await reorderDepots(newItems.map((d) => d.id));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler beim Sortieren");
      load();
    }
  };

  return (
    <>
      <div className="page-header">
        <h2>Depots</h2>
        <button className="btn btn-primary" onClick={openCreate}>
          + Neues Depot
        </button>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {loading ? (
        <div className="loading">Laden…</div>
      ) : depots.length === 0 ? (
        <div className="card empty-state">
          <div className="empty-icon">📊</div>
          <p>Noch keine Depots vorhanden.</p>
          <p style={{ fontSize: "0.9rem", color: "#666" }}>
            Erstellen Sie ein virtuelles Depot, um mehrere Konten zu
            aggregieren und Zinsannahmen für die Projektion zu konfigurieren.
          </p>
        </div>
      ) : (
        <div className="card table-wrapper">
          <table>
            <thead>
              <tr>
                <th style={{ width: 40 }}></th>
                <th>Name</th>
                <th>Zinssatz (p.a.)</th>
                <th>Konten</th>
                <th>Aktueller Wert</th>
                <th style={{ width: 120 }}>Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {depots.map((d, index) => (
                <tr
                  key={d.id}
                  className={dragOverIndex === index ? "drag-over" : ""}
                  draggable
                  onDragStart={(e) => handleDragStart(e, d.id)}
                  onDragEnd={handleDragEnd}
                  onDragEnter={(e) => handleDragEnter(e, index)}
                  onDragLeave={handleDragLeave}
                  onDragOver={handleDragOver}
                  onDrop={(e) => handleDrop(e, index)}
                >
                  <td className="drag-handle">⠿</td>
                  <td>{d.name}</td>
                  <td>{d.interestRate.toFixed(2)} %</td>
                  <td>
                    <div className="tag-list">
                      {d.accounts?.map((a) => (
                        <span key={a.id} className="tag">
                          {a.name}
                        </span>
                      ))}
                      {(!d.accounts || d.accounts.length === 0) && (
                        <span style={{ color: "#999", fontSize: "0.85rem" }}>
                          Keine Konten
                        </span>
                      )}
                    </div>
                  </td>
                  <td>
                    <span className="amount">
                      {formatCurrency(getDepotBalance(d))} €
                    </span>
                  </td>
                  <td>
                    <div className="actions-cell">
                      <button
                        className="btn btn-sm btn-secondary"
                        onClick={() => openEdit(d)}
                      >
                        ✏️
                      </button>
                      <button
                        className="btn btn-sm btn-danger"
                        onClick={() => handleDelete(d)}
                      >
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
            <h3>{editing ? "Depot bearbeiten" : "Neues Depot"}</h3>
            <form onSubmit={handleSubmit}>
              <div className="form-group">
                <label>Name</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  autoFocus
                  placeholder="Depotname"
                />
              </div>
              <div className="form-group">
                <label>Zinssatz p.a. (%)</label>
                <input
                  type="number"
                  step="0.01"
                  value={form.interestRate}
                  onChange={(e) =>
                    setForm({ ...form, interestRate: e.target.value })
                  }
                  placeholder="z.B. 5.0"
                />
              </div>
              {accounts.length > 0 && (
                <div className="form-group">
                  <label>Zugeordnete Konten</label>
                  <div className="multi-select">
                    {accounts.map((a) => (
                      <label key={a.id}>
                        <input
                          type="checkbox"
                          checked={form.accountIds.includes(a.id)}
                          onChange={() => toggleAccount(a.id)}
                        />
                        {a.name}{" "}
                        <span style={{ color: "#888", fontSize: "0.85rem" }}>
                          ({formatCurrency(a.balance)} {a.currency})
                        </span>
                      </label>
                    ))}
                  </div>
                </div>
              )}
              {formError && <div className="form-error">{formError}</div>}
              <div className="form-actions">
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => setShowForm(false)}
                >
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
