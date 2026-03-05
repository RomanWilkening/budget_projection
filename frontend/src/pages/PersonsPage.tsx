import { useEffect, useState, type FormEvent } from "react";
import type { Person } from "../types";
import { getPersons, createPerson, updatePerson, deletePerson } from "../api";

export default function PersonsPage() {
  const [persons, setPersons] = useState<Person[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Person | null>(null);
  const [formName, setFormName] = useState("");
  const [formError, setFormError] = useState("");

  const load = () => {
    setLoading(true);
    getPersons()
      .then(setPersons)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const openCreate = () => {
    setEditing(null);
    setFormName("");
    setFormError("");
    setShowForm(true);
  };

  const openEdit = (p: Person) => {
    setEditing(p);
    setFormName(p.name);
    setFormError("");
    setShowForm(true);
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    const name = formName.trim();
    if (!name) {
      setFormError("Name ist erforderlich.");
      return;
    }
    try {
      if (editing) {
        await updatePerson(editing.id, { name });
      } else {
        await createPerson({ name });
      }
      setShowForm(false);
      load();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Fehler");
    }
  };

  const handleDelete = async (p: Person) => {
    if (!confirm(`"${p.name}" wirklich löschen?`)) return;
    try {
      await deletePerson(p.id);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler");
    }
  };

  return (
    <>
      <div className="page-header">
        <h2>Personen</h2>
        <button className="btn btn-primary" onClick={openCreate}>
          + Neue Person
        </button>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {loading ? (
        <div className="loading">Laden…</div>
      ) : persons.length === 0 ? (
        <div className="card empty-state">
          <div className="empty-icon">👤</div>
          <p>Noch keine Personen vorhanden.</p>
        </div>
      ) : (
        <div className="card table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Erstellt</th>
                <th style={{ width: 120 }}>Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {persons.map((p) => (
                <tr key={p.id}>
                  <td>{p.name}</td>
                  <td>{new Date(p.createdAt).toLocaleDateString("de-DE")}</td>
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
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showForm && (
        <div className="form-overlay" onClick={() => setShowForm(false)}>
          <div className="form-modal" onClick={(e) => e.stopPropagation()}>
            <h3>{editing ? "Person bearbeiten" : "Neue Person"}</h3>
            <form onSubmit={handleSubmit}>
              <div className="form-group">
                <label>Name</label>
                <input
                  type="text"
                  value={formName}
                  onChange={(e) => setFormName(e.target.value)}
                  autoFocus
                  placeholder="Name eingeben"
                />
                {formError && <div className="form-error">{formError}</div>}
              </div>
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
