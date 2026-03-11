import { useState, useEffect } from "react";
import axios from "axios";

const API_BASE_URL = "http://localhost:8000";

function App() {
  // Theme state
  const [theme, setTheme] = useState(() => {
    return localStorage.getItem("gizzard-theme") || "dark";
  });

  // Create User form state
  const [createForm, setCreateForm] = useState({
    id: "",
    name: "",
    email: "",
    shard: "auto",
  });
  const [createResult, setCreateResult] = useState(null);
  const [createError, setCreateError] = useState(null);
  const [createLoading, setCreateLoading] = useState(false);

  // Fetch User form state
  const [fetchId, setFetchId] = useState("");
  const [fetchResult, setFetchResult] = useState(null);
  const [fetchError, setFetchError] = useState(null);
  const [fetchLoading, setFetchLoading] = useState(false);

  // All Users state
  const [allUsers, setAllUsers] = useState([]);
  const [allUsersLoading, setAllUsersLoading] = useState(false);

  // Shard status
  const [shardStatus, setShardStatus] = useState([]);
  const [shardLoading, setShardLoading] = useState(false);

  // Algorithm states
  const [activeAlgoTab, setActiveAlgoTab] = useState("clocks");
  const [clocksData, setClocksData] = useState(null);
  const [clocksLoading, setClocksLoading] = useState(false);
  const [eventsData, setEventsData] = useState(null);
  const [snapshotResult, setSnapshotResult] = useState(null);
  const [snapshotLoading, setSnapshotLoading] = useState(false);
  const [snapshotStates, setSnapshotStates] = useState(null);
  const [snapshotStatesLoading, setSnapshotStatesLoading] = useState(false);
  const [electionResult, setElectionResult] = useState(null);
  const [electionLoading, setElectionLoading] = useState(false);
  const [leaderStatus, setLeaderStatus] = useState(null);
  const [leaderLoading, setLeaderLoading] = useState(false);
  const [hashRingStatus, setHashRingStatus] = useState(null);
  const [hashRingLoading, setHashRingLoading] = useState(false);
  const [lookupKey, setLookupKey] = useState("");
  const [lookupResult, setLookupResult] = useState(null);
  const [lookupLoading, setLookupLoading] = useState(false);

  // Apply theme to document
  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
    localStorage.setItem("gizzard-theme", theme);
  }, [theme]);

  const toggleTheme = () => {
    setTheme(theme === "dark" ? "light" : "dark");
  };

  // Load shard status and users on mount
  useEffect(() => {
    loadShardStatus();
    loadAllUsers();
  }, []);

  // Calculate auto-assigned shard based on user ID
  const getAutoShard = (userId) => {
    if (!userId) return null;
    return ((parseInt(userId) - 1) % 4) + 1;
  };

  // Get the effective shard (manual override or auto)
  const getEffectiveShard = () => {
    if (createForm.shard === "auto") {
      return getAutoShard(createForm.id);
    }
    return parseInt(createForm.shard);
  };

  const loadShardStatus = async () => {
    setShardLoading(true);
    try {
      const response = await axios.get(`${API_BASE_URL}/shards`);
      setShardStatus(response.data.shards || []);
    } catch (err) {
      console.error("Failed to load shard status:", err);
    } finally {
      setShardLoading(false);
    }
  };

  const loadAllUsers = async () => {
    setAllUsersLoading(true);
    try {
      const response = await axios.get(`${API_BASE_URL}/users`);
      setAllUsers(response.data.users || []);
    } catch (err) {
      console.error("Failed to load users:", err);
    } finally {
      setAllUsersLoading(false);
    }
  };

  const handleCreateUser = async (e) => {
    e.preventDefault();
    setCreateLoading(true);
    setCreateError(null);
    setCreateResult(null);

    const effectiveShard = getEffectiveShard();

    try {
      const response = await axios.post(`${API_BASE_URL}/users`, {
        id: parseInt(createForm.id),
        name: createForm.name,
        email: createForm.email,
        shard_id: effectiveShard,
      });
      setCreateResult(response.data);
      setCreateForm({ id: "", name: "", email: "", shard: "auto" });
      // Refresh data
      loadShardStatus();
      loadAllUsers();
    } catch (err) {
      setCreateError(err.response?.data?.error || err.message);
    } finally {
      setCreateLoading(false);
    }
  };

  const handleFetchUser = async (e) => {
    e.preventDefault();
    setFetchLoading(true);
    setFetchError(null);
    setFetchResult(null);

    try {
      const response = await axios.get(`${API_BASE_URL}/users/${fetchId}`);
      setFetchResult(response.data);
    } catch (err) {
      if (err.response?.status === 404) {
        setFetchError("User not found");
      } else {
        setFetchError(err.response?.data?.error || err.message);
      }
    } finally {
      setFetchLoading(false);
    }
  };

  const getShardColor = (shardId) => {
    const colors = ["#3b82f6", "#10b981", "#f59e0b", "#ef4444"];
    return colors[(shardId - 1) % colors.length];
  };

  // =============================================
  // Algorithm Handlers
  // =============================================

  const fetchClocks = async () => {
    setClocksLoading(true);
    try {
      const [clocksRes, eventsRes] = await Promise.all([
        axios.get(`${API_BASE_URL}/clocks`),
        axios.get(`${API_BASE_URL}/events`),
      ]);
      setClocksData(clocksRes.data);
      setEventsData(eventsRes.data);
    } catch (err) {
      console.error("Failed to fetch clocks:", err);
    } finally {
      setClocksLoading(false);
    }
  };

  const triggerSnapshot = async () => {
    setSnapshotLoading(true);
    setSnapshotResult(null);
    try {
      const response = await axios.post(`${API_BASE_URL}/snapshot`, {});
      setSnapshotResult(response.data);
      // After a short delay, fetch snapshot states
      setTimeout(fetchSnapshotStates, 1000);
    } catch (err) {
      setSnapshotResult({ error: err.response?.data?.error || err.message });
    } finally {
      setSnapshotLoading(false);
    }
  };

  const fetchSnapshotStates = async () => {
    setSnapshotStatesLoading(true);
    try {
      const response = await axios.get(`${API_BASE_URL}/snapshot`);
      setSnapshotStates(response.data);
    } catch (err) {
      console.error("Failed to fetch snapshot states:", err);
    } finally {
      setSnapshotStatesLoading(false);
    }
  };

  const triggerElection = async () => {
    setElectionLoading(true);
    setElectionResult(null);
    try {
      const response = await axios.post(`${API_BASE_URL}/election/start`, {});
      setElectionResult(response.data);
      // After a short delay, fetch leader status
      setTimeout(fetchLeaderStatus, 1500);
    } catch (err) {
      setElectionResult({ error: err.response?.data?.error || err.message });
    } finally {
      setElectionLoading(false);
    }
  };

  const fetchLeaderStatus = async () => {
    setLeaderLoading(true);
    try {
      const response = await axios.get(`${API_BASE_URL}/election/leader`);
      setLeaderStatus(response.data);
    } catch (err) {
      console.error("Failed to fetch leader status:", err);
    } finally {
      setLeaderLoading(false);
    }
  };

  const fetchHashRingStatus = async () => {
    setHashRingLoading(true);
    try {
      const response = await axios.get(`${API_BASE_URL}/hash-ring/status`);
      setHashRingStatus(response.data);
    } catch (err) {
      console.error("Failed to fetch hash ring:", err);
    } finally {
      setHashRingLoading(false);
    }
  };

  const lookupHashKey = async (e) => {
    e.preventDefault();
    if (!lookupKey) return;
    setLookupLoading(true);
    setLookupResult(null);
    try {
      const payload = /^\d+$/.test(lookupKey)
        ? { user_id: parseInt(lookupKey) }
        : { key: lookupKey };
      const response = await axios.post(`${API_BASE_URL}/hash-ring/lookup`, payload);
      setLookupResult(response.data);
    } catch (err) {
      setLookupResult({ error: err.response?.data?.error || err.message });
    } finally {
      setLookupLoading(false);
    }
  };

  const algoTabs = [
    { id: "clocks", label: "⏱ Vector Clocks", color: "#3b82f6" },
    { id: "snapshot", label: "📸 Snapshot", color: "#10b981" },
    { id: "election", label: "👑 Election", color: "#f59e0b" },
    { id: "hashing", label: "🔗 Hashing", color: "#8b5cf6" },
  ];

  return (
    <div className="app">
      <header className="header">
        <button
          className="theme-toggle"
          onClick={toggleTheme}
          aria-label="Toggle theme"
        >
          {theme === "dark" ? (
            <svg viewBox="0 0 24 24" fill="none" className="theme-icon">
              <circle
                cx="12"
                cy="12"
                r="5"
                stroke="currentColor"
                strokeWidth="2"
              />
              <path
                d="M12 1V3M12 21V23M4.22 4.22L5.64 5.64M18.36 18.36L19.78 19.78M1 12H3M21 12H23M4.22 19.78L5.64 18.36M18.36 5.64L19.78 4.22"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
            </svg>
          ) : (
            <svg viewBox="0 0 24 24" fill="none" className="theme-icon">
              <path
                d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          )}
        </button>
        <div className="logo">
          <svg viewBox="0 0 24 24" fill="none" className="logo-icon">
            <path
              d="M12 2L2 7L12 12L22 7L12 2Z"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
            <path
              d="M2 17L12 22L22 17"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
            <path
              d="M2 12L12 17L22 12"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
          <span>Gizzard</span>
        </div>
        <h1>Distributed Database Sharding Framework</h1>
        <p>
          High-performance data distribution across multiple shards with
          automatic load balancing
        </p>
      </header>

      <div className="container">
        {/* Shard Status */}
        <section className="section shard-status">
          <div className="section-header">
            <h2>Shard Status</h2>
            <button
              onClick={loadShardStatus}
              disabled={shardLoading}
              className="refresh-btn"
            >
              <svg viewBox="0 0 24 24" fill="none" className="refresh-icon">
                <path
                  d="M1 4V10H7"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                <path
                  d="M23 20V14H17"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                <path
                  d="M20.49 9A9 9 0 0 0 5.64 5.64L1 10M23 14L18.36 18.36A9 9 0 0 1 3.51 15"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
              {shardLoading ? "Loading..." : "Refresh"}
            </button>
          </div>
          <div className="shard-grid">
            {shardStatus.map((shard) => (
              <div
                key={shard.shard_id}
                className={`shard-card ${shard.status}`}
                style={{ borderColor: getShardColor(shard.shard_id) }}
              >
                <div
                  className="shard-header"
                  style={{ backgroundColor: getShardColor(shard.shard_id) }}
                >
                  <span className="shard-icon">⬡</span>
                  Shard {shard.shard_id}
                </div>
                <div className="shard-body">
                  <div className="shard-stat">
                    <span className="stat-label">Status</span>
                    <span className={`stat-value status-${shard.status}`}>
                      {shard.status === "online" ? "● Online" : "○ Offline"}
                    </span>
                  </div>
                  <div className="shard-stat">
                    <span className="stat-label">IP Address</span>
                    <span className="stat-value ip-address">
                      {shard.host || "localhost"}
                    </span>
                  </div>
                  <div className="shard-stat">
                    <span className="stat-label">Port</span>
                    <span className="stat-value">{shard.port}</span>
                  </div>
                  <div className="shard-stat">
                    <span className="stat-label">Users</span>
                    <span className="stat-value">{shard.user_count}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </section>

        <div className="forms-container">
          {/* Create User Section */}
          <section className="section">
            <h2>Create User</h2>
            <form onSubmit={handleCreateUser} className="form">
              <div className="form-group">
                <label>User ID</label>
                <input
                  type="number"
                  value={createForm.id}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, id: e.target.value })
                  }
                  placeholder="Enter User ID"
                  required
                  min="1"
                />
              </div>
              <div className="form-group">
                <label>Name</label>
                <input
                  type="text"
                  value={createForm.name}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, name: e.target.value })
                  }
                  placeholder="Enter Name"
                  required
                />
              </div>
              <div className="form-group">
                <label>Email</label>
                <input
                  type="email"
                  value={createForm.email}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, email: e.target.value })
                  }
                  placeholder="Enter Email"
                  required
                />
              </div>
              <div className="form-group">
                <label>Shard Assignment</label>
                <div className="shard-selector">
                  <select
                    value={createForm.shard}
                    onChange={(e) =>
                      setCreateForm({ ...createForm, shard: e.target.value })
                    }
                    className="shard-select"
                  >
                    <option value="auto">Auto-assign (Recommended)</option>
                    <option value="1">Shard 1</option>
                    <option value="2">Shard 2</option>
                    <option value="3">Shard 3</option>
                    <option value="4">Shard 4</option>
                  </select>
                </div>
                <small className="shard-hint">
                  {createForm.shard === "auto" ? (
                    <>
                      Auto-assigned to{" "}
                      <strong
                        style={{
                          color: getShardColor(
                            getAutoShard(createForm.id) || 1,
                          ),
                        }}
                      >
                        Shard{" "}
                        {createForm.id ? getAutoShard(createForm.id) : "?"}
                      </strong>{" "}
                      based on User ID
                    </>
                  ) : (
                    <>
                      Manually assigned to{" "}
                      <strong
                        style={{
                          color: getShardColor(parseInt(createForm.shard)),
                        }}
                      >
                        Shard {createForm.shard}
                      </strong>
                    </>
                  )}
                </small>
              </div>
              <button
                type="submit"
                disabled={createLoading}
                className="submit-btn"
              >
                {createLoading ? (
                  <>
                    <span className="spinner"></span>
                    Creating...
                  </>
                ) : (
                  "Create User"
                )}
              </button>
            </form>

            {createResult && (
              <div className="result success">
                <div className="result-header">
                  <svg
                    viewBox="0 0 24 24"
                    fill="none"
                    className="result-icon success"
                  >
                    <path
                      d="M22 11.08V12a10 10 0 1 1-5.93-9.14"
                      stroke="currentColor"
                      strokeWidth="2"
                    />
                    <polyline
                      points="22 4 12 14.01 9 11.01"
                      stroke="currentColor"
                      strokeWidth="2"
                    />
                  </svg>
                  <h4>User Created Successfully!</h4>
                </div>
                <div className="result-body">
                  <p>
                    <strong>ID:</strong> {createResult.user?.id}
                  </p>
                  <p>
                    <strong>Name:</strong> {createResult.user?.name}
                  </p>
                  <p>
                    <strong>Email:</strong> {createResult.user?.email}
                  </p>
                  <p
                    className="shard-info"
                    style={{ color: getShardColor(createResult.shard_id) }}
                  >
                    Stored in Shard {createResult.shard_id}
                  </p>
                </div>
              </div>
            )}

            {createError && (
              <div className="result error">
                <div className="result-header">
                  <svg
                    viewBox="0 0 24 24"
                    fill="none"
                    className="result-icon error"
                  >
                    <circle
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="2"
                    />
                    <line
                      x1="15"
                      y1="9"
                      x2="9"
                      y2="15"
                      stroke="currentColor"
                      strokeWidth="2"
                    />
                    <line
                      x1="9"
                      y1="9"
                      x2="15"
                      y2="15"
                      stroke="currentColor"
                      strokeWidth="2"
                    />
                  </svg>
                  <h4>Error</h4>
                </div>
                <p>{createError}</p>
              </div>
            )}
          </section>

          {/* Fetch User Section */}
          <section className="section">
            <h2>Fetch User</h2>
            <form onSubmit={handleFetchUser} className="form">
              <div className="form-group">
                <label>User ID</label>
                <input
                  type="number"
                  value={fetchId}
                  onChange={(e) => setFetchId(e.target.value)}
                  placeholder="Enter User ID to fetch"
                  required
                  min="1"
                />
                <small>
                  Will query{" "}
                  <strong
                    style={{ color: getShardColor(getAutoShard(fetchId) || 1) }}
                  >
                    Shard {fetchId ? getAutoShard(fetchId) : "?"}
                  </strong>
                </small>
              </div>
              <button
                type="submit"
                disabled={fetchLoading}
                className="submit-btn secondary"
              >
                {fetchLoading ? (
                  <>
                    <span className="spinner"></span>
                    Fetching...
                  </>
                ) : (
                  "Fetch User"
                )}
              </button>
            </form>

            {fetchResult && (
              <div className="result success">
                <div className="result-header">
                  <svg
                    viewBox="0 0 24 24"
                    fill="none"
                    className="result-icon success"
                  >
                    <path
                      d="M22 11.08V12a10 10 0 1 1-5.93-9.14"
                      stroke="currentColor"
                      strokeWidth="2"
                    />
                    <polyline
                      points="22 4 12 14.01 9 11.01"
                      stroke="currentColor"
                      strokeWidth="2"
                    />
                  </svg>
                  <h4>User Found!</h4>
                </div>
                <div className="result-body">
                  <p>
                    <strong>ID:</strong> {fetchResult.user?.id}
                  </p>
                  <p>
                    <strong>Name:</strong> {fetchResult.user?.name}
                  </p>
                  <p>
                    <strong>Email:</strong> {fetchResult.user?.email}
                  </p>
                  <p
                    className="shard-info"
                    style={{ color: getShardColor(fetchResult.shard_id) }}
                  >
                    Retrieved from Shard {fetchResult.shard_id}
                  </p>
                </div>
              </div>
            )}

            {fetchError && (
              <div className="result error">
                <div className="result-header">
                  <svg
                    viewBox="0 0 24 24"
                    fill="none"
                    className="result-icon error"
                  >
                    <circle
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="2"
                    />
                    <line
                      x1="15"
                      y1="9"
                      x2="9"
                      y2="15"
                      stroke="currentColor"
                      strokeWidth="2"
                    />
                    <line
                      x1="9"
                      y1="9"
                      x2="15"
                      y2="15"
                      stroke="currentColor"
                      strokeWidth="2"
                    />
                  </svg>
                  <h4>Error</h4>
                </div>
                <p>{fetchError}</p>
              </div>
            )}
          </section>
        </div>

        {/* All Users Section */}
        <section className="section users-section">
          <div className="section-header">
            <h2>All Users</h2>
            <button
              onClick={loadAllUsers}
              disabled={allUsersLoading}
              className="refresh-btn"
            >
              <svg viewBox="0 0 24 24" fill="none" className="refresh-icon">
                <path
                  d="M1 4V10H7"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                <path
                  d="M23 20V14H17"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                <path
                  d="M20.49 9A9 9 0 0 0 5.64 5.64L1 10M23 14L18.36 18.36A9 9 0 0 1 3.51 15"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
              {allUsersLoading ? "Loading..." : "Refresh"}
            </button>
          </div>

          {allUsers.length === 0 ? (
            <div className="empty-state">
              <svg viewBox="0 0 24 24" fill="none" className="empty-icon">
                <path
                  d="M17 21V19C17 17.9391 16.5786 16.9217 15.8284 16.1716C15.0783 15.4214 14.0609 15 13 15H5C3.93913 15 2.92172 15.4214 2.17157 16.1716C1.42143 16.9217 1 17.9391 1 19V21"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                <path
                  d="M9 11C11.2091 11 13 9.20914 13 7C13 4.79086 11.2091 3 9 3C6.79086 3 5 4.79086 5 7C5 9.20914 6.79086 11 9 11Z"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                <path
                  d="M23 21V19C22.9993 18.1137 22.7044 17.2528 22.1614 16.5523C21.6184 15.8519 20.8581 15.3516 20 15.13"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                <path
                  d="M16 3.13C16.8604 3.35031 17.623 3.85071 18.1676 4.55232C18.7122 5.25392 19.0078 6.11683 19.0078 7.005C19.0078 7.89318 18.7122 8.75608 18.1676 9.45769C17.623 10.1593 16.8604 10.6597 16 10.88"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
              <p>No users found</p>
              <span>Create some users using the form above!</span>
            </div>
          ) : (
            <div className="users-table-container">
              <table className="users-table">
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Name</th>
                    <th>Email</th>
                    <th>Shard</th>
                  </tr>
                </thead>
                <tbody>
                  {allUsers.map((user) => (
                    <tr key={user.id}>
                      <td>{user.id}</td>
                      <td>{user.name}</td>
                      <td>{user.email}</td>
                      <td>
                        <span
                          className="shard-badge"
                          style={{
                            backgroundColor: getShardColor(user.shard_id),
                          }}
                        >
                          Shard {user.shard_id}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {/* =============================================
            ALGORITHMS DASHBOARD
            ============================================= */}
        <section className="section algo-dashboard">
          <div className="section-header">
            <h2>🧪 Distributed Algorithms Dashboard</h2>
          </div>
          <p className="algo-subtitle">
            Trigger and visualize distributed systems algorithms running across all shard nodes
          </p>

          {/* Algorithm Tabs */}
          <div className="algo-tabs">
            {algoTabs.map((tab) => (
              <button
                key={tab.id}
                className={`algo-tab ${activeAlgoTab === tab.id ? "active" : ""}`}
                onClick={() => setActiveAlgoTab(tab.id)}
                style={activeAlgoTab === tab.id ? { borderColor: tab.color, color: tab.color } : {}}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Vector Clocks Panel */}
          {activeAlgoTab === "clocks" && (
            <div className="algo-panel">
              <div className="algo-info">
                <h3>⏱ Lamport / Vector Clocks</h3>
                <p>Each node maintains a vector of logical timestamps to track causal ordering of events. Vector clocks tick automatically on every user operation (create/read).</p>
              </div>
              <button
                onClick={fetchClocks}
                disabled={clocksLoading}
                className="submit-btn algo-trigger-btn"
                style={{ background: "linear-gradient(135deg, #3b82f6, #2563eb)" }}
              >
                {clocksLoading ? (
                  <><span className="spinner"></span> Fetching...</>
                ) : (
                  "📡 Fetch Vector Clocks"
                )}
              </button>

              {clocksData && (
                <div className="algo-results">
                  <h4>Node Clocks</h4>
                  <div className="clock-grid">
                    {clocksData.clocks?.map((clock, idx) => (
                      <div key={idx} className="clock-card">
                        <div className="clock-node-id">{clock.node_id || `Node ${idx + 1}`}</div>
                        <div className="clock-vector">
                          {clock.vector_clock && Object.entries(clock.vector_clock).map(([node, val]) => (
                            <div key={node} className="clock-entry">
                              <span className="clock-key">{node}</span>
                              <span className="clock-val">{val}</span>
                            </div>
                          ))}
                        </div>
                        <div className="clock-event-count">
                          {clock.event_count || 0} events logged
                        </div>
                      </div>
                    ))}
                  </div>

                  {eventsData && eventsData.events && eventsData.events.length > 0 && (
                    <div className="events-section">
                      <h4>Recent Events ({eventsData.event_count})</h4>
                      <div className="events-list">
                        {eventsData.events.slice(-10).reverse().map((event, idx) => (
                          <div key={idx} className="event-item">
                            <span className="event-type">{event.event_type}</span>
                            <span className="event-desc">{event.description}</span>
                            <span className="event-node">{event.node_id}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Chandy-Lamport Snapshot Panel */}
          {activeAlgoTab === "snapshot" && (
            <div className="algo-panel">
              <div className="algo-info">
                <h3>📸 Chandy-Lamport Snapshot</h3>
                <p>Captures a consistent global snapshot across all nodes without stopping the system. Uses marker messages to record local states and channel states between nodes.</p>
              </div>
              <div className="algo-btn-row">
                <button
                  onClick={triggerSnapshot}
                  disabled={snapshotLoading}
                  className="submit-btn algo-trigger-btn"
                  style={{ background: "linear-gradient(135deg, #10b981, #059669)" }}
                >
                  {snapshotLoading ? (
                    <><span className="spinner"></span> Capturing...</>
                  ) : (
                    "📸 Take Global Snapshot"
                  )}
                </button>
                <button
                  onClick={fetchSnapshotStates}
                  disabled={snapshotStatesLoading}
                  className="refresh-btn"
                >
                  {snapshotStatesLoading ? "Loading..." : "🔄 Refresh States"}
                </button>
              </div>

              {snapshotResult && (
                <div className="algo-results">
                  <h4>Snapshot Initiated</h4>
                  <div className="algo-result-card success">
                    <p><strong>Snapshot ID:</strong> {snapshotResult.snapshot_id}</p>
                    <p><strong>Initiated from:</strong> Shard {snapshotResult.initiated_at}</p>
                    <p><strong>Message:</strong> {snapshotResult.message || snapshotResult.error}</p>
                  </div>
                </div>
              )}

              {snapshotStates && (
                <div className="algo-results">
                  <h4>Snapshot States from All Nodes</h4>
                  <div className="snapshot-grid">
                    {snapshotStates.snapshots?.map((snap, idx) => (
                      <div key={idx} className="snapshot-card">
                        <div className="snapshot-node">{snap.node_id || `Node ${idx + 1}`}</div>
                        {snap.snapshot ? (
                          <div className="snapshot-details">
                            <p><strong>Status:</strong> {snap.snapshot.completed ? "✅ Complete" : "⏳ In Progress"}</p>
                            <p><strong>Users:</strong> {snap.snapshot.local_state?.user_count ?? "N/A"}</p>
                            <p><strong>Recorded at:</strong> {snap.snapshot.recorded_at ? new Date(snap.snapshot.recorded_at).toLocaleTimeString() : "N/A"}</p>
                          </div>
                        ) : (
                          <div className="snapshot-details">
                            <p>Snapshots: {snap.snapshots ? Object.keys(snap.snapshots).length : 0}</p>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Leader Election Panel */}
          {activeAlgoTab === "election" && (
            <div className="algo-panel">
              <div className="algo-info">
                <h3>👑 Bully Leader Election</h3>
                <p>The Bully algorithm elects a coordinator node. When triggered, nodes with lower IDs send ELECTION messages to higher-ID nodes. If no higher node responds, the sender declares itself leader. The highest-ID alive node always wins.</p>
              </div>
              <div className="algo-btn-row">
                <button
                  onClick={triggerElection}
                  disabled={electionLoading}
                  className="submit-btn algo-trigger-btn"
                  style={{ background: "linear-gradient(135deg, #f59e0b, #d97706)" }}
                >
                  {electionLoading ? (
                    <><span className="spinner"></span> Electing...</>
                  ) : (
                    "👑 Trigger Election"
                  )}
                </button>
                <button
                  onClick={fetchLeaderStatus}
                  disabled={leaderLoading}
                  className="refresh-btn"
                >
                  {leaderLoading ? "Loading..." : "🔄 Check Leader"}
                </button>
              </div>

              {electionResult && (
                <div className="algo-results">
                  <h4>Election Triggered</h4>
                  <div className="algo-result-card success">
                    <p><strong>From Node:</strong> {electionResult.from_node}</p>
                    <p><strong>Message:</strong> {electionResult.message || electionResult.error}</p>
                  </div>
                </div>
              )}

              {leaderStatus && (
                <div className="algo-results">
                  <h4>Leader Status (All Nodes)</h4>
                  <div className="election-grid">
                    {leaderStatus.nodes?.map((node, idx) => {
                      const state = node.state || {};
                      return (
                        <div key={idx} className={`election-card ${state.is_leader ? "is-leader" : ""}`}>
                          <div className="election-node">
                            Node {state.node_id || node.node_id}
                            {state.is_leader && <span className="leader-crown">👑</span>}
                          </div>
                          <div className="election-details">
                            <p><strong>Leader:</strong> Node {state.current_leader}</p>
                            <p><strong>Is Leader:</strong> {state.is_leader ? "Yes" : "No"}</p>
                            <p><strong>Term:</strong> {state.election_term}</p>
                            <p><strong>Electing:</strong> {state.election_active ? "Yes" : "No"}</p>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Consistent Hashing Panel */}
          {activeAlgoTab === "hashing" && (
            <div className="algo-panel">
              <div className="algo-info">
                <h3>🔗 Consistent Hashing</h3>
                <p>Uses a hash ring with 150 virtual nodes per physical node. Keys are mapped to the nearest clockwise node on the ring. Adding/removing nodes only redistributes ~1/N of the keys.</p>
              </div>
              <div className="algo-btn-row">
                <button
                  onClick={fetchHashRingStatus}
                  disabled={hashRingLoading}
                  className="submit-btn algo-trigger-btn"
                  style={{ background: "linear-gradient(135deg, #8b5cf6, #7c3aed)" }}
                >
                  {hashRingLoading ? (
                    <><span className="spinner"></span> Loading...</>
                  ) : (
                    "🔗 View Hash Ring"
                  )}
                </button>
              </div>

              {hashRingStatus && (
                <div className="algo-results">
                  <h4>Hash Ring Status</h4>
                  <div className="hashring-stats">
                    <div className="hashring-stat">
                      <span className="stat-label">Physical Nodes</span>
                      <span className="stat-value">{hashRingStatus.hash_ring?.nodes?.length || 0}</span>
                    </div>
                    <div className="hashring-stat">
                      <span className="stat-label">Virtual Nodes/Node</span>
                      <span className="stat-value">{hashRingStatus.hash_ring?.virtual_nodes_per_node || 0}</span>
                    </div>
                    <div className="hashring-stat">
                      <span className="stat-label">Total VNodes</span>
                      <span className="stat-value">{hashRingStatus.hash_ring?.total_virtual_nodes || 0}</span>
                    </div>
                  </div>

                  {hashRingStatus.hash_ring?.key_distribution_pct && (
                    <div className="distribution-section">
                      <h4>Key Distribution</h4>
                      <div className="distribution-bars">
                        {Object.entries(hashRingStatus.hash_ring.key_distribution_pct).map(([node, pct]) => (
                          <div key={node} className="dist-bar-row">
                            <span className="dist-label">{node}</span>
                            <div className="dist-bar-bg">
                              <div
                                className="dist-bar-fill"
                                style={{ width: `${pct}%`, background: getShardColor(parseInt(node.replace(/\D/g, '')) || 1) }}
                              ></div>
                            </div>
                            <span className="dist-pct">{pct.toFixed(1)}%</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {/* Key Lookup */}
              <div className="algo-lookup">
                <h4>🔍 Key Lookup</h4>
                <form onSubmit={lookupHashKey} className="lookup-form">
                  <input
                    type="text"
                    value={lookupKey}
                    onChange={(e) => setLookupKey(e.target.value)}
                    placeholder="Enter user ID or key (e.g. 42 or my_key)"
                    className="lookup-input"
                  />
                  <button
                    type="submit"
                    disabled={lookupLoading}
                    className="submit-btn secondary"
                    style={{ minWidth: "120px" }}
                  >
                    {lookupLoading ? "..." : "Lookup"}
                  </button>
                </form>

                {lookupResult && !lookupResult.error && (
                  <div className="algo-result-card success">
                    <p><strong>Key:</strong> {lookupResult.key}</p>
                    <p><strong>Assigned Node:</strong> <span style={{ color: getShardColor(parseInt(String(lookupResult.assigned_node).replace(/\D/g, '')) || 1), fontWeight: 700 }}>{lookupResult.assigned_node}</span></p>
                    {lookupResult.comparison && (
                      <p><strong>Comparison:</strong> {lookupResult.comparison}</p>
                    )}
                    <p><strong>Key Hash:</strong> <code>{lookupResult.key_hash}</code></p>
                  </div>
                )}
              </div>
            </div>
          )}
        </section>

        {/* Architecture Diagram */}
        <section className="section architecture">
          <h2>System Architecture</h2>
          <div className="architecture-content">
            <pre className="diagram">
              {`
              ┌─────────────────────────────────────┐
              │       Client (React Frontend)       │
              └─────────────────┬───────────────────┘
                                │
                                ▼
              ┌─────────────────────────────────────┐
              │    API Gateway (Port 8000)          │
              │    ├── Request Routing              │
              │    ├── Load Balancing               │
              │    ├── Consistent Hashing           │
              │    └── Algorithm Orchestration      │
              └─────────────────┬───────────────────┘
                                │
        ┌───────────┬───────────┼───────────┬───────────┐
        │           │           │           │           │
        ▼           ▼           ▼           ▼           │
   ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐      │
   │ Shard 1 │ │ Shard 2 │ │ Shard 3 │ │ Shard 4 │      │
   │  :8001  │ │  :8002  │ │  :8003  │ │  :8004  │      │
   │ VClock  │ │ VClock  │ │ VClock  │ │ VClock  │      │
   │ Snap    │ │ Snap    │ │ Snap    │ │ Snap    │      │
   │ Bully   │ │ Bully   │ │ Bully   │ │ Bully   │      │
   └─────────┘ └─────────┘ └─────────┘ └─────────┘      │
`}
            </pre>
            <div className="formula-box">
              <h4>Sharding Formula</h4>
              <code>shard_id = ((user_id - 1) % 4) + 1</code>
              <div className="formula-examples">
                <span>User 1 → Shard 1</span>
                <span>User 2 → Shard 2</span>
                <span>User 3 → Shard 3</span>
                <span>User 4 → Shard 4</span>
                <span>User 5 → Shard 1</span>
                <span>...</span>
              </div>
              <h4 style={{ marginTop: "16px" }}>Algorithms</h4>
              <div className="formula-examples">
                <span>⏱ Vector Clocks</span>
                <span>📸 Chandy-Lamport</span>
                <span>👑 Bully Election</span>
                <span>🔗 Consistent Hash</span>
              </div>
            </div>
          </div>
        </section>

        <footer className="footer">
          <p>Gizzard Distributed Database Sharding Framework</p>
          <span>Built with Go, Gin, SQLite & React</span>
        </footer>
      </div>
    </div>
  );
}

export default App;

