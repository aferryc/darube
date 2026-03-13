import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.jsx'
import './index.css'

const platform = globalThis?.darube?.platform;
if (platform === 'darwin') {
  document.documentElement.classList.add('platform-mac');
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
