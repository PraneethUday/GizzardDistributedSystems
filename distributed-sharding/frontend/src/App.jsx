import { useState, useEffect } from 'react'
import axios from 'axios'

const API_BASE_URL = 'http://localhost:8000'

function App() {
  // Create User form state
  const [createForm, setCreateForm] = useState({
    id: '',
    name: '',
    email: ''
  })
  const [createResult, setCreateResult] = useState(null)
  const [createError, setCreateError] = useState(null)
  const [createLoading, setCreateLoading] = useState(false)

  // Fetch User form state
  const [fetchId, setFetchId] = useState('')
  const [fetchResult, setFetchResult] = useState(null)
  const [fetchError, setFetchError] = useState(null)
  const [fetchLoading, setFetchLoading] = useState(false)

  // All Users state
  const [allUsers, setAllUsers] = useState([])
  const [allUsersLoading, setAllUsersLoading] = useState(false)

  // Shard status
  const [shardStatus, setShardStatus] = useState([])
  const [shardLoading, setShardLoading] = useState(false)

  // Load shard status and users on mount
  useEffect(() => {
    loadShardStatus()
    loadAllUsers()
  }, [])

  const loadShardStatus = async () => {
    setShardLoading(true)
    try {
      const response = await axios.get(`${API_BASE_URL}/shards`)
      setShardStatus(response.data.shards || [])
    } catch (err) {
      console.error('Failed to load shard status:', err)
    } finally {
      setShardLoading(false)
    }
  }

  const loadAllUsers = async () => {
    setAllUsersLoading(true)
    try {
      const response = await axios.get(`${API_BASE_URL}/users`)
      setAllUsers(response.data.users || [])
    } catch (err) {
      console.error('Failed to load users:', err)
    } finally {
      setAllUsersLoading(false)
    }
  }

  const handleCreateUser = async (e) => {
    e.preventDefault()
    setCreateLoading(true)
    setCreateError(null)
    setCreateResult(null)

    try {
      const response = await axios.post(`${API_BASE_URL}/users`, {
        id: parseInt(createForm.id),
        name: createForm.name,
        email: createForm.email
      })
      setCreateResult(response.data)
      setCreateForm({ id: '', name: '', email: '' })
      // Refresh data
      loadShardStatus()
      loadAllUsers()
    } catch (err) {
      setCreateError(err.response?.data?.error || err.message)
    } finally {
      setCreateLoading(false)
    }
  }

  const handleFetchUser = async (e) => {
    e.preventDefault()
    setFetchLoading(true)
    setFetchError(null)
    setFetchResult(null)

    try {
      const response = await axios.get(`${API_BASE_URL}/users/${fetchId}`)
      setFetchResult(response.data)
    } catch (err) {
      if (err.response?.status === 404) {
        setFetchError('User not found')
      } else {
        setFetchError(err.response?.data?.error || err.message)
      }
    } finally {
      setFetchLoading(false)
    }
  }

  const getShardColor = (shardId) => {
    const colors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444']
    return colors[(shardId - 1) % colors.length]
  }

  return (
    <div className="app">
      <header className="header">
        <h1>Distributed Sharding Demo</h1>
        <p>A distributed database system with 4 shards</p>
      </header>

      <div className="container">
        {/* Shard Status */}
        <section className="section shard-status">
          <h2>Shard Status</h2>
          <button onClick={loadShardStatus} disabled={shardLoading} className="refresh-btn">
            {shardLoading ? 'Loading...' : 'Refresh'}
          </button>
          <div className="shard-grid">
            {shardStatus.map((shard) => (
              <div
                key={shard.shard_id}
                className={`shard-card ${shard.status}`}
                style={{ borderColor: getShardColor(shard.shard_id) }}
              >
                <div className="shard-header" style={{ backgroundColor: getShardColor(shard.shard_id) }}>
                  Shard {shard.shard_id}
                </div>
                <div className="shard-body">
                  <p><strong>Status:</strong> {shard.status}</p>
                  <p><strong>Port:</strong> {shard.port}</p>
                  <p><strong>Users:</strong> {shard.user_count}</p>
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
                <label>User ID:</label>
                <input
                  type="number"
                  value={createForm.id}
                  onChange={(e) => setCreateForm({ ...createForm, id: e.target.value })}
                  placeholder="Enter User ID"
                  required
                  min="1"
                />
                <small>Shard assignment: User {createForm.id || '?'} → Shard {createForm.id ? ((parseInt(createForm.id) - 1) % 4) + 1 : '?'}</small>
              </div>
              <div className="form-group">
                <label>Name:</label>
                <input
                  type="text"
                  value={createForm.name}
                  onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                  placeholder="Enter Name"
                  required
                />
              </div>
              <div className="form-group">
                <label>Email:</label>
                <input
                  type="email"
                  value={createForm.email}
                  onChange={(e) => setCreateForm({ ...createForm, email: e.target.value })}
                  placeholder="Enter Email"
                  required
                />
              </div>
              <button type="submit" disabled={createLoading} className="submit-btn">
                {createLoading ? 'Creating...' : 'Create User'}
              </button>
            </form>

            {createResult && (
              <div className="result success">
                <h4>User Created Successfully!</h4>
                <p><strong>ID:</strong> {createResult.user?.id}</p>
                <p><strong>Name:</strong> {createResult.user?.name}</p>
                <p><strong>Email:</strong> {createResult.user?.email}</p>
                <p className="shard-info" style={{ color: getShardColor(createResult.shard_id) }}>
                  Stored in Shard {createResult.shard_id}
                </p>
              </div>
            )}

            {createError && (
              <div className="result error">
                <h4>Error</h4>
                <p>{createError}</p>
              </div>
            )}
          </section>

          {/* Fetch User Section */}
          <section className="section">
            <h2>Fetch User</h2>
            <form onSubmit={handleFetchUser} className="form">
              <div className="form-group">
                <label>User ID:</label>
                <input
                  type="number"
                  value={fetchId}
                  onChange={(e) => setFetchId(e.target.value)}
                  placeholder="Enter User ID to fetch"
                  required
                  min="1"
                />
                <small>Will query Shard {fetchId ? ((parseInt(fetchId) - 1) % 4) + 1 : '?'}</small>
              </div>
              <button type="submit" disabled={fetchLoading} className="submit-btn">
                {fetchLoading ? 'Fetching...' : 'Fetch User'}
              </button>
            </form>

            {fetchResult && (
              <div className="result success">
                <h4>User Found!</h4>
                <p><strong>ID:</strong> {fetchResult.user?.id}</p>
                <p><strong>Name:</strong> {fetchResult.user?.name}</p>
                <p><strong>Email:</strong> {fetchResult.user?.email}</p>
                <p className="shard-info" style={{ color: getShardColor(fetchResult.shard_id) }}>
                  Retrieved from Shard {fetchResult.shard_id}
                </p>
              </div>
            )}

            {fetchError && (
              <div className="result error">
                <h4>Error</h4>
                <p>{fetchError}</p>
              </div>
            )}
          </section>
        </div>

        {/* All Users Section */}
        <section className="section users-section">
          <h2>All Users</h2>
          <button onClick={loadAllUsers} disabled={allUsersLoading} className="refresh-btn">
            {allUsersLoading ? 'Loading...' : 'Refresh'}
          </button>
          
          {allUsers.length === 0 ? (
            <p className="no-users">No users found. Create some users above!</p>
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
                          style={{ backgroundColor: getShardColor(user.shard_id) }}
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

        {/* Architecture Diagram */}
        <section className="section architecture">
          <h2>Architecture</h2>
          <pre className="diagram">
{`
              Client (React App)
                     |
                     v
            API Gateway (port 8000)
                     |
      ┌──────┬──────┼──────┬──────┐
      v      v      v      v      
   Shard1  Shard2  Shard3  Shard4
   :8001   :8002   :8003   :8004
   
   Formula: shard = ((userID - 1) % 4) + 1
   
   User 1 → Shard 1    User 5 → Shard 1
   User 2 → Shard 2    User 6 → Shard 2
   User 3 → Shard 3    User 7 → Shard 3
   User 4 → Shard 4    User 8 → Shard 4
`}
          </pre>
        </section>
      </div>
    </div>
  )
}

export default App
