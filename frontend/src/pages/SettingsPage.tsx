import { useEffect, useState, type FormEvent } from "react";
import type { BridgeStatus } from "../types";
import { getSettings, updateSettings, getBankingBridgeStatus } from "../api";

export default function SettingsPage() {
  const [bankingBridgeUrl, setBankingBridgeUrl] = useState("");
  const [savedUrl, setSavedUrl] = useState("");
  const [bridgeStatus, setBridgeStatus] = useState<BridgeStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    setLoading(true);
    getSettings()
      .then((settings) => {
        const url = settings.banking_bridge_url || "";
        setBankingBridgeUrl(url);
        setSavedUrl(url);
        if (url) {
          loadBridgeStatus();
        }
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const loadBridgeStatus = () => {
    getBankingBridgeStatus()
      .then(setBridgeStatus)
      .catch(() => setBridgeStatus(null));
  };

  const handleSave = async (e: FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await updateSettings({ banking_bridge_url: bankingBridgeUrl.trim() });
      setSavedUrl(bankingBridgeUrl.trim());
      setSuccess("Einstellungen gespeichert.");
      if (bankingBridgeUrl.trim()) {
        loadBridgeStatus();
      } else {
        setBridgeStatus(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fehler beim Speichern");
    } finally {
      setSaving(false);
    }
  };

  const handleTestConnection = async () => {
    setTesting(true);
    setError("");
    setSuccess("");
    try {
      // Save first if URL changed
      if (bankingBridgeUrl.trim() !== savedUrl) {
        await updateSettings({ banking_bridge_url: bankingBridgeUrl.trim() });
        setSavedUrl(bankingBridgeUrl.trim());
      }
      const status = await getBankingBridgeStatus();
      setBridgeStatus(status);
      if (status.connected) {
        setSuccess("Verbindung erfolgreich!");
      } else {
        setError(status.error || "Verbindung fehlgeschlagen.");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Verbindungstest fehlgeschlagen");
    } finally {
      setTesting(false);
    }
  };

  const hasChanges = bankingBridgeUrl.trim() !== savedUrl;

  return (
    <>
      <div className="page-header">
        <h2>Einstellungen</h2>
      </div>

      {loading ? (
        <div className="loading">Laden…</div>
      ) : (
        <>
          <div className="card" style={{ padding: "1.5rem" }}>
            <h3 style={{ marginTop: 0, marginBottom: "1rem" }}>🔗 Banking Bridge</h3>
            <p style={{ color: "#666", fontSize: "0.9rem", marginBottom: "1rem" }}>
              Verbinden Sie die Anwendung mit Ihrem Banking Bridge Service, um Kontostände
              automatisch zu synchronisieren und wiederkehrende Transaktionen zu analysieren.
            </p>

            <form onSubmit={handleSave}>
              <div className="form-group">
                <label>Banking Bridge URL</label>
                <input
                  type="url"
                  value={bankingBridgeUrl}
                  onChange={(e) => {
                    setBankingBridgeUrl(e.target.value);
                    setSuccess("");
                  }}
                  placeholder="z.B. http://192.168.1.100:8080"
                  style={{ maxWidth: "500px" }}
                />
              </div>

              {error && <div className="form-error" style={{ marginBottom: "1rem" }}>{error}</div>}
              {success && (
                <div style={{ color: "#2e7d32", marginBottom: "1rem", fontSize: "0.9rem" }}>
                  ✅ {success}
                </div>
              )}

              <div className="form-actions" style={{ justifyContent: "flex-start", gap: "0.5rem" }}>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={saving || !hasChanges}
                >
                  {saving ? "⏳ Speichern…" : "Speichern"}
                </button>
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={handleTestConnection}
                  disabled={testing || !bankingBridgeUrl.trim()}
                >
                  {testing ? "⏳ Teste…" : "🔌 Verbindung testen"}
                </button>
              </div>
            </form>
          </div>

          {bridgeStatus && bankingBridgeUrl.trim() && (
            <div className="card" style={{ padding: "1rem", marginTop: "1rem" }}>
              <h4 style={{ marginTop: 0, marginBottom: "0.75rem" }}>Verbindungsstatus</h4>
              <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem", fontSize: "0.9rem" }}>
                <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                  <span>{bridgeStatus.configured ? "🟢" : "⚪"}</span>
                  <span>Konfiguriert: {bridgeStatus.configured ? "Ja" : "Nein"}</span>
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                  <span>{bridgeStatus.connected ? "🟢" : "🔴"}</span>
                  <span>Verbunden: {bridgeStatus.connected ? "Ja" : "Nein"}</span>
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                  <span>🌐</span>
                  <span>URL: {bridgeStatus.url}</span>
                </div>
                {bridgeStatus.error && (
                  <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", color: "#c62828" }}>
                    <span>⚠️</span>
                    <span>{bridgeStatus.error}</span>
                  </div>
                )}
              </div>
            </div>
          )}
        </>
      )}
    </>
  );
}
