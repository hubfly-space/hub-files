import './index.css'
import './App.css'

function App() {
  return (
    <div className="main-container">
      <div className="header">
        <div className="breadcrumb">
          <span>/</span>
          <span>projects</span>
          <span>/</span>
          <span>website</span>
          <span className="active">●</span>
        </div>
        <div className="actions">
          <button title="Upload">⬆</button>
          <button title="New">＋</button>
          <button title="Refresh">⟳</button>
          <button title="View Toggle">⧉</button>
        </div>
      </div>
      <div className="content">
        <div className="empty-state">
          This folder is empty
        </div>
      </div>
    </div>
  )
}

export default App
