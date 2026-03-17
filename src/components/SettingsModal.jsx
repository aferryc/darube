import React, { useEffect, useState } from 'react';

export function SettingsModal({ show, onClose, apiUrl, layoutDirection, onLayoutChange }) {
  const [activeTab, setActiveTab] = useState('common');
  const [settings, setSettings] = useState({ layout_direction: 'vertical', teleport_profiles: [] });
  const [loading, setLoading] = useState(false);
  const [profileForm, setProfileForm] = useState(null); // { id?, name, cluster, user, profile } for add/edit
  const [profileError, setProfileError] = useState('');
  const [detecting, setDetecting] = useState(false);

  useEffect(() => {
    if (!show) return;
    setActiveTab('common');
    setProfileForm(null);
    setProfileError('');
    fetch(`${apiUrl}/api/settings`)
      .then(r => r.json())
      .then(data => {
        if (data.success !== false && data.layout_direction) {
          setSettings({
            layout_direction: data.layout_direction || 'vertical',
            teleport_profiles: data.teleport_profiles || [],
          });
        }
      })
      .catch(() => {});
  }, [show, apiUrl]);

  const saveLayout = async (dir) => {
    setLoading(true);
    try {
      const res = await fetch(`${apiUrl}/api/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ layout_direction: dir }),
      });
      const data = await res.json();
      if (data.success) {
        setSettings(s => ({ ...s, layout_direction: dir }));
        onLayoutChange?.(dir);
      }
    } finally {
      setLoading(false);
    }
  };

  const saveProfile = async () => {
    if (!profileForm?.name?.trim()) {
      setProfileError('Name is required');
      return;
    }
    setProfileError('');
    setLoading(true);
    try {
      const url = profileForm.id
        ? `${apiUrl}/api/settings/teleport-profiles/${profileForm.id}`
        : `${apiUrl}/api/settings/teleport-profiles`;
      const method = profileForm.id ? 'PUT' : 'POST';
      const body = {
        name: profileForm.name.trim(),
        cluster: profileForm.cluster?.trim() || '',
        user: profileForm.user?.trim() || '',
        profile: profileForm.profile?.trim() || '',
      };
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (data.success) {
        const list = await fetch(`${apiUrl}/api/settings`).then(r => r.json());
        if (list.teleport_profiles) {
          setSettings(s => ({ ...s, teleport_profiles: list.teleport_profiles }));
        }
        setProfileForm(null);
      } else {
        setProfileError(data.error || 'Failed to save');
      }
    } catch (err) {
      setProfileError(err.message || 'Failed to save');
    } finally {
      setLoading(false);
    }
  };

  const autoDetect = async () => {
    if (!apiUrl) return;
    setDetecting(true);
    setProfileError('');
    try {
      const res = await fetch(`${apiUrl}/api/teleport/detect`);
      const data = await res.json();
      if (data.success) {
        setProfileForm(p => ({
          ...(p || { name: '', cluster: '', user: '', profile: '' }),
          cluster: data.cluster || '',
          user: data.user || '',
          profile: data.profile || '',
          ...(p?.name ? {} : { name: data.cluster || 'Detected' }),
        }));
      } else {
        setProfileError(data.error || 'Could not detect tsh profile');
      }
    } catch (err) {
      setProfileError(err.message || 'Detection failed');
    } finally {
      setDetecting(false);
    }
  };

  const deleteProfile = async (id) => {
    if (!window.confirm('Delete this Teleport profile?')) return;
    setLoading(true);
    try {
      const res = await fetch(`${apiUrl}/api/settings/teleport-profiles/${id}`, { method: 'DELETE' });
      const data = await res.json();
      if (data.success) {
        setSettings(s => ({
          ...s,
          teleport_profiles: s.teleport_profiles.filter(p => p.id !== id),
        }));
        if (profileForm?.id === id) setProfileForm(null);
      }
    } finally {
      setLoading(false);
    }
  };

  if (!show) return null;

  return (
    <div className="settings-view">
      <div className="modal-header">
        <div className="modal-header-left">
          <h3>Settings</h3>
        </div>
        <button type="button" className="btn-icon" onClick={onClose} style={{ padding: 4 }}>✕</button>
      </div>

        <div className="modal-tabs" style={{ marginBottom: 16 }}>
          {['common', 'teleport'].map(tab => (
            <button
              key={tab}
              type="button"
              className={`modal-tab ${activeTab === tab ? 'active' : ''}`}
              onClick={() => setActiveTab(tab)}
            >
              {tab === 'common' ? 'Common' : 'Teleport'}
            </button>
          ))}
        </div>

      <div className="settings-body">
        {activeTab === 'common' && (
          <div className="form-group">
            <label>View as</label>
            <div style={{ display: 'flex', gap: 12, marginTop: 8 }}>
              <label className="form-checkbox-label" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <input
                  type="radio"
                  name="layout"
                  checked={settings.layout_direction === 'vertical'}
                  onChange={() => saveLayout('vertical')}
                  disabled={loading}
                />
                Vertical
              </label>
              <label className="form-checkbox-label" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <input
                  type="radio"
                  name="layout"
                  checked={settings.layout_direction === 'horizontal'}
                  onChange={() => saveLayout('horizontal')}
                  disabled={loading}
                />
                Horizontal
              </label>
            </div>
            <div className="form-help-text" style={{ marginTop: 8 }}>
              Layout of query editor and results pane.
            </div>
          </div>
        )}

        {activeTab === 'teleport' && (
          <div className="settings-section">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
              <span className="form-help-text" style={{ margin: 0 }}>Reusable Teleport profiles (cluster, user, profile)</span>
              {!profileForm && (
                <button type="button" className="secondary" onClick={() => setProfileForm({ name: '', cluster: '', user: '', profile: '' })} style={{ height: 30, padding: '0 12px' }}>
                  + Add Profile
                </button>
              )}
            </div>

            {profileForm && (
              <div className="form-group" style={{ marginBottom: 16, padding: 12, background: 'rgba(255,255,255,0.03)', borderRadius: 8, border: '1px solid var(--border)' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
                  <strong>{profileForm.id ? 'Edit Profile' : 'New Profile'}</strong>
                  <button type="button" className="secondary" onClick={() => { setProfileForm(null); setProfileError(''); }} style={{ padding: '2px 8px' }}>Cancel</button>
                </div>
                <div className="form-row">
                  <div className="form-group flex-1">
                    <label>Name</label>
                    <input value={profileForm.name} onChange={e => setProfileForm(p => ({ ...p, name: e.target.value }))} placeholder="e.g. Production" />
                  </div>
                </div>
                <div className="form-row">
                  <div className="form-group flex-1">
                    <label>Cluster</label>
                    <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                      <input value={profileForm.cluster} onChange={e => setProfileForm(p => ({ ...p, cluster: e.target.value }))} placeholder="example-teleport-cluster" style={{ flex: 1 }} />
                      <button type="button" className="secondary" onClick={autoDetect} disabled={detecting} style={{ height: 34, padding: '0 12px', whiteSpace: 'nowrap' }}>
                        {detecting ? 'Detecting…' : 'Auto-detect'}
                      </button>
                    </div>
                    <div className="form-help-text" style={{ marginTop: 4 }}>
                      Fills cluster, user, and profile from current <code>tsh</code> session.
                    </div>
                  </div>
                </div>
                <div className="form-row">
                  <div className="form-group flex-1">
                    <label>User (optional)</label>
                    <input value={profileForm.user} onChange={e => setProfileForm(p => ({ ...p, user: e.target.value }))} placeholder="alice" />
                  </div>
                  <div className="form-group flex-1">
                    <label>Profile (optional)</label>
                    <input value={profileForm.profile} onChange={e => setProfileForm(p => ({ ...p, profile: e.target.value }))} placeholder="default" />
                  </div>
                </div>
                {profileError && <div className="api-error" style={{ marginBottom: 8 }}>{profileError}</div>}
                <button type="button" onClick={saveProfile} disabled={loading} style={{ marginTop: 8 }}>Save Profile</button>
              </div>
            )}

            <table className="api-kv-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Cluster</th>
                  <th>User</th>
                  <th>Profile</th>
                  <th style={{ width: 90 }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {settings.teleport_profiles.length === 0 && !profileForm && (
                  <tr><td colSpan={5} style={{ color: 'var(--text-muted)', fontStyle: 'italic', padding: 16 }}>No Teleport profiles yet. Add one above.</td></tr>
                )}
                {settings.teleport_profiles.map(p => (
                  <tr key={p.id}>
                    <td>{p.name}</td>
                    <td>{p.cluster || '—'}</td>
                    <td>{p.user || '—'}</td>
                    <td>{p.profile || '—'}</td>
                    <td>
                      <button type="button" className="secondary" onClick={() => setProfileForm({ id: p.id, name: p.name, cluster: p.cluster || '', user: p.user || '', profile: p.profile || '' })} style={{ padding: '2px 8px', marginRight: 4 }}>Edit</button>
                      <button type="button" className="secondary" onClick={() => deleteProfile(p.id)} disabled={loading} style={{ padding: '2px 8px' }}>Delete</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

      </div>
    </div>
  );
}
