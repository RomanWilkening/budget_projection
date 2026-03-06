import { Routes, Route, NavLink } from "react-router-dom";
import "./App.css";
import DashboardPage from "./pages/DashboardPage";
import PersonsPage from "./pages/PersonsPage";
import AccountsPage from "./pages/AccountsPage";
import PositionsPage from "./pages/PositionsPage";

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
          <NavLink to="/personen">
            <span className="nav-icon">👤</span> Personen
          </NavLink>
          <NavLink to="/konten">
            <span className="nav-icon">🏦</span> Konten
          </NavLink>
          <NavLink to="/positionen">
            <span className="nav-icon">📋</span> Positionen
          </NavLink>
        </nav>
      </aside>
      <main className="main-content">
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/personen" element={<PersonsPage />} />
          <Route path="/konten" element={<AccountsPage />} />
          <Route path="/positionen" element={<PositionsPage />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;

