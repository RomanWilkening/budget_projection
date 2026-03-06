import { Routes, Route, NavLink } from "react-router-dom";
import "./App.css";
import DashboardPage from "./pages/DashboardPage";
import PersonsPage from "./pages/PersonsPage";
import AccountsPage from "./pages/AccountsPage";
import PositionsPage from "./pages/PositionsPage";
import ProjectionPage from "./pages/ProjectionPage";
import BankingBridgePage from "./pages/BankingBridgePage";
import SettingsPage from "./pages/SettingsPage";

function App() {
  return (
    <div className="app-layout">
      <aside className="sidebar">
        <div className="sidebar-header">
          <h1>Budget Planer</h1>
          <p>Finanzplanung</p>
        </div>
        <nav className="sidebar-nav">
          <NavLink to="/" end>
            <span className="nav-icon">📊</span> Übersicht
          </NavLink>
          <NavLink to="/projektion">
            <span className="nav-icon">📈</span> Projektion
          </NavLink>
          <NavLink to="/personen">
            <span className="nav-icon">👤</span> Personen
          </NavLink>
          <NavLink to="/konten">
            <span className="nav-icon">🏦</span> Konten
          </NavLink>
          <NavLink to="/positionen">
            <span className="nav-icon">📋</span> Positionen
          </NavLink>
          <NavLink to="/banking-bridge">
            <span className="nav-icon">🔗</span> Banking Bridge
          </NavLink>
          <NavLink to="/einstellungen">
            <span className="nav-icon">⚙️</span> Einstellungen
          </NavLink>
        </nav>
      </aside>
      <main className="main-content">
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/projektion" element={<ProjectionPage />} />
          <Route path="/personen" element={<PersonsPage />} />
          <Route path="/konten" element={<AccountsPage />} />
          <Route path="/positionen" element={<PositionsPage />} />
          <Route path="/banking-bridge" element={<BankingBridgePage />} />
          <Route path="/einstellungen" element={<SettingsPage />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;

