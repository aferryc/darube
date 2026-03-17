import React, { useEffect, useState } from 'react';

export function SettingsModal({ show, onClose, apiUrl, layoutDirection, onLayoutChange, settings: externalSettings, onSettingsChange }) {
  const [activeTab, setActiveTab] = useState('common');
  const [settings, setSettings] = useState({
    layout_direction: 'vertical',
    teleport_profiles: [],
    global_script_timeout_ms: 0,
    global_query_timeout_ms: 0,
    global_api_timeout_ms: 0,
    max_lines_query: 0,
    max_lines_script: 0,
    max_lines_body: 0,
    theme_variant: '',
    ui_theme_custom: '',
    ui_font_family: '',
    ui_font_size: 0,
    ui_font_color: '',
    ui_text_primary: '',
    ui_text_muted: '',
    ui_text_accent: '',
  });
  const [loading, setLoading] = useState(false);
  const [profileForm, setProfileForm] = useState(null); // { id?, name, cluster, user, profile } for add/edit
  const [profileError, setProfileError] = useState('');
  const [detecting, setDetecting] = useState(false);
  const defaultFontSize = 13;
  const displayFontSize = settings.ui_font_size && Number(settings.ui_font_size) > 0
    ? Number(settings.ui_font_size)
    : defaultFontSize;
  const isFontSizeDefault = !(settings.ui_font_size && Number(settings.ui_font_size) > 0);

  useEffect(() => {
    if (!show) return;
    setActiveTab('common');
    setProfileForm(null);
    setProfileError('');
    fetch(`${apiUrl}/api/settings`)
      .then(r => r.json())
      .then(data => {
        if (data && data.success !== false) {
          setSettings({
            layout_direction: data.layout_direction || 'vertical',
            teleport_profiles: data.teleport_profiles || [],
            global_script_timeout_ms: data.global_script_timeout_ms ?? 0,
            global_query_timeout_ms: data.global_query_timeout_ms ?? 0,
            global_api_timeout_ms: data.global_api_timeout_ms ?? 0,
            max_lines_query: data.max_lines_query ?? 0,
            max_lines_script: data.max_lines_script ?? 0,
            max_lines_body: data.max_lines_body ?? 0,
            theme_variant: data.theme_variant || '',
            ui_theme_custom: data.ui_theme_custom || '',
            ui_font_family: data.ui_font_family || '',
            ui_font_size: data.ui_font_size || 0,
            ui_font_color: data.ui_font_color || '',
            ui_text_primary: data.ui_text_primary || '',
            ui_text_muted: data.ui_text_muted || '',
            ui_text_accent: data.ui_text_accent || '',
          });
          onSettingsChange?.(data);
        }
      })
      .catch(() => {});
  }, [show, apiUrl]);

  // Keep local state loosely in sync when parent settings change (e.g. from elsewhere).
  useEffect(() => {
    if (!externalSettings) return;
    setSettings(s => ({
      ...s,
      ...externalSettings,
    }));
  }, [externalSettings]);

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
        onSettingsChange?.({ ...settings, layout_direction: dir });
      }
    } finally {
      setLoading(false);
    }
  };

  const saveTimeoutsAndLimits = async () => {
    setLoading(true);
    try {
      const body = {
        global_script_timeout_ms: Number(settings.global_script_timeout_ms) || 0,
        global_query_timeout_ms: Number(settings.global_query_timeout_ms) || 0,
        global_api_timeout_ms: Number(settings.global_api_timeout_ms) || 0,
        max_lines_query: Number(settings.max_lines_query) || 0,
        max_lines_script: Number(settings.max_lines_script) || 0,
        max_lines_body: Number(settings.max_lines_body) || 0,
      };
      const res = await fetch(`${apiUrl}/api/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (data.success) {
        onSettingsChange?.({ ...settings, ...body });
      }
    } finally {
      setLoading(false);
    }
  };

  const saveTheme = async () => {
    setLoading(true);
    try {
      const body = {
        theme_variant: settings.theme_variant || '',
        ui_theme_custom: settings.ui_theme_custom || '',
        ui_font_family: settings.ui_font_family || '',
        ui_font_size: Number(settings.ui_font_size) || 0,
        ui_font_color: settings.ui_font_color || '',
        ui_text_primary: settings.ui_text_primary || '',
        ui_text_muted: settings.ui_text_muted || '',
        ui_text_accent: settings.ui_text_accent || '',
      };
      const res = await fetch(`${apiUrl}/api/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (data.success) {
        onSettingsChange?.({ ...settings, ...body });
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
      <div className="modal-header settings-header">
        <div className="modal-header-left">
          <h3>Settings</h3>
          <span className="settings-subtitle">Preferences and system limits for this workspace.</span>
        </div>
        <button type="button" className="btn-icon close-btn" onClick={onClose} aria-label="Close settings">✕</button>
      </div>

      <div className="modal-tabs settings-tabs">
        {['common', 'teleport', 'theme'].map(tab => (
          <button
            key={tab}
            type="button"
            className={`modal-tab ${activeTab === tab ? 'active' : ''}`}
            onClick={() => setActiveTab(tab)}
          >
            {tab === 'common' ? 'Common' : tab === 'teleport' ? 'Teleport' : 'Theme'}
          </button>
        ))}
      </div>

      <div className="settings-body">
        {activeTab === 'common' && (
          <div className="settings-grid">
            <div className="settings-card">
              <div className="settings-card-title">Layout</div>
              <div className="settings-card-body">
                <div className="choice-row">
                  <label className={`choice-pill ${settings.layout_direction === 'vertical' ? 'active' : ''}`}>
                    <input
                      type="radio"
                      name="layout"
                      checked={settings.layout_direction === 'vertical'}
                      onChange={() => saveLayout('vertical')}
                      disabled={loading}
                    />
                    <span>Vertical</span>
                  </label>
                  <label className={`choice-pill ${settings.layout_direction === 'horizontal' ? 'active' : ''}`}>
                    <input
                      type="radio"
                      name="layout"
                      checked={settings.layout_direction === 'horizontal'}
                      onChange={() => saveLayout('horizontal')}
                      disabled={loading}
                    />
                    <span>Horizontal</span>
                  </label>
                </div>
                <div className="form-help-text compact">
                  Layout of query editor and results pane.
                </div>
              </div>
            </div>

            <div className="settings-card">
              <div className="settings-card-title">Timeout Limits</div>
              <div className="settings-card-body settings-fields cols-3">
                <div className="form-group">
                  <label>Script Timeout Max (ms)</label>
                  <input
                    type="number"
                    value={settings.global_script_timeout_ms}
                    onChange={e => setSettings(s => ({ ...s, global_script_timeout_ms: e.target.value }))}
                    placeholder="15000"
                  />
                </div>
                <div className="form-group">
                  <label>Query Timeout Max (ms)</label>
                  <input
                    type="number"
                    value={settings.global_query_timeout_ms}
                    onChange={e => setSettings(s => ({ ...s, global_query_timeout_ms: e.target.value }))}
                    placeholder="30000"
                  />
                </div>
                <div className="form-group">
                  <label>API Request Timeout Max (ms)</label>
                  <input
                    type="number"
                    value={settings.global_api_timeout_ms}
                    onChange={e => setSettings(s => ({ ...s, global_api_timeout_ms: e.target.value }))}
                    placeholder="30000"
                  />
                </div>
              </div>
              <div className="settings-card-note">Tip: use 0 for unlimited.</div>
            </div>

            <div className="settings-card">
              <div className="settings-card-title">Line Limits</div>
              <div className="settings-card-body settings-fields cols-3">
                <div className="form-group">
                  <label>Max Query Lines</label>
                  <input
                    type="number"
                    value={settings.max_lines_query}
                    onChange={e => setSettings(s => ({ ...s, max_lines_query: e.target.value }))}
                    placeholder="0 = no limit"
                  />
                </div>
                <div className="form-group">
                  <label>Max Script Lines</label>
                  <input
                    type="number"
                    value={settings.max_lines_script}
                    onChange={e => setSettings(s => ({ ...s, max_lines_script: e.target.value }))}
                    placeholder="0 = no limit"
                  />
                </div>
                <div className="form-group">
                  <label>Max Body Lines</label>
                  <input
                    type="number"
                    value={settings.max_lines_body}
                    onChange={e => setSettings(s => ({ ...s, max_lines_body: e.target.value }))}
                    placeholder="0 = no limit"
                  />
                </div>
              </div>
              <div className="settings-card-note">Tip: use 0 for unlimited.</div>
            </div>

            <div className="modal-footer settings-footer">
              <button type="button" className="secondary" onClick={onClose}>Close</button>
              <button type="button" onClick={saveTimeoutsAndLimits} disabled={loading}>
                Save
              </button>
            </div>
          </div>
        )}

        {activeTab === 'teleport' && (
          <div className="settings-section">
            <div className="settings-card">
              <div className="settings-card-header">
                <div>
                  <div className="settings-card-title">Teleport Profiles</div>
                  <div className="settings-card-subtitle">Reusable profiles for cluster, user, and profile selection.</div>
                </div>
                {!profileForm && (
                  <button type="button" className="secondary" onClick={() => setProfileForm({ name: '', cluster: '', user: '', profile: '' })}>
                    + Add Profile
                  </button>
                )}
              </div>

            {profileForm && (
              <div className="settings-form-card">
                <div className="settings-form-header">
                  <strong>{profileForm.id ? 'Edit Profile' : 'New Profile'}</strong>
                  <button type="button" className="secondary" onClick={() => { setProfileForm(null); setProfileError(''); }}>Cancel</button>
                </div>
                <div className="settings-fields cols-2">
                  <div className="form-group">
                    <label>Name</label>
                    <input value={profileForm.name} onChange={e => setProfileForm(p => ({ ...p, name: e.target.value }))} placeholder="e.g. Production" />
                  </div>
                  <div className="form-group span-2">
                    <label>Cluster</label>
                    <div className="form-file-input-row">
                      <input value={profileForm.cluster} onChange={e => setProfileForm(p => ({ ...p, cluster: e.target.value }))} placeholder="example-teleport-cluster" style={{ flex: 1 }} />
                      <button type="button" className="secondary" onClick={autoDetect} disabled={detecting}>
                        {detecting ? 'Detecting…' : 'Auto-detect'}
                      </button>
                    </div>
                    <div className="form-help-text compact">
                      Fills cluster, user, and profile from current <code>tsh</code> session.
                    </div>
                  </div>
                  <div className="form-group">
                    <label>User (optional)</label>
                    <input value={profileForm.user} onChange={e => setProfileForm(p => ({ ...p, user: e.target.value }))} placeholder="alice" />
                  </div>
                  <div className="form-group">
                    <label>Profile (optional)</label>
                    <input value={profileForm.profile} onChange={e => setProfileForm(p => ({ ...p, profile: e.target.value }))} placeholder="default" />
                  </div>
                </div>
                {profileError && <div className="api-error" style={{ marginBottom: 8 }}>{profileError}</div>}
                <div className="settings-form-actions">
                  <button type="button" onClick={saveProfile} disabled={loading}>Save Profile</button>
                </div>
              </div>
            )}

              <div className="settings-table-card">
                <table className="api-kv-table">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Cluster</th>
                      <th>User</th>
                      <th>Profile</th>
                      <th style={{ width: 120 }}>Actions</th>
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
                          <button type="button" className="secondary" onClick={() => setProfileForm({ id: p.id, name: p.name, cluster: p.cluster || '', user: p.user || '', profile: p.profile || '' })}>Edit</button>
                          <button type="button" className="secondary" onClick={() => deleteProfile(p.id)} disabled={loading}>Delete</button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'theme' && (
          <div className="settings-grid">
            <div className="settings-card">
              <div className="settings-card-title">Live Preview</div>
              <div className="settings-preview">
                <div
                  className="settings-preview-title"
                  style={{
                    fontFamily: settings.ui_font_family || 'var(--app-font-family)',
                    fontSize: displayFontSize > 0 ? `${displayFontSize + 4}px` : 'calc(var(--app-font-size) + 4px)',
                    color: settings.ui_text_primary || settings.ui_font_color || 'var(--text-main)',
                  }}
                >
                  Darube Workspace
                </div>
                <div
                  className="settings-preview-body"
                  style={{
                    fontFamily: settings.ui_font_family || 'var(--app-font-family)',
                    fontSize: displayFontSize > 0 ? `${displayFontSize}px` : 'var(--app-font-size)',
                    color: settings.ui_text_muted || 'var(--text-muted)',
                  }}
                >
                  UI text preview. Applies to labels, tables, and modal content.
                </div>
                <div className="settings-preview-chip">
                  <span className="mono" style={{ color: settings.ui_text_accent || 'var(--accent)' }}>SELECT</span> *{" "}
                  <span className="muted">FROM</span> connections;
                </div>
              </div>
            </div>

            <div className="settings-card">
              <div className="settings-card-title">Color Scheme</div>
              <div className="settings-card-body">
                <div className="form-group">
                  <label>Preset</label>
                  <select
                    value={settings.theme_variant}
                    onChange={e => setSettings(s => ({ ...s, theme_variant: e.target.value }))}
                  >
                    <option value="">Default</option>
                    <option value="dark">Dark</option>
                    <option value="ocean">Ocean</option>
                    <option value="high-contrast">High Contrast</option>
                  </select>
                  <div className="form-help-text compact">
                    Basic presets. Advanced overrides can be provided via custom theme JSON below.
                  </div>
                </div>
              </div>
            </div>

            <div className="settings-card">
              <div className="settings-card-title">Typography</div>
              <div className="settings-card-body">
                <div className="form-group">
                  <label>Font Family</label>
                  <input
                    value={settings.ui_font_family}
                    onChange={e => setSettings(s => ({ ...s, ui_font_family: e.target.value }))}
                    placeholder='e.g. "SF Pro Text", system-ui'
                  />
                  <div className="form-help-text compact">Applies to UI labels, tables, and modals. Editor code font is unchanged.</div>
                  <div className="settings-pill-row">
                    <button type="button" className="secondary mini-btn" onClick={() => setSettings(s => ({ ...s, ui_font_family: '"Plus Jakarta Sans", system-ui' }))}>Jakarta Sans</button>
                    <button type="button" className="secondary mini-btn" onClick={() => setSettings(s => ({ ...s, ui_font_family: '"JetBrains Mono", monospace' }))}>JetBrains Mono</button>
                    <button type="button" className="secondary mini-btn" onClick={() => setSettings(s => ({ ...s, ui_font_family: 'system-ui' }))}>System UI</button>
                  </div>
                </div>

                <div className="settings-fields cols-2">
                  <div className="form-group">
                    <label>Font Size (px)</label>
                    <div className="settings-inline">
                      <input
                        type="number"
                        value={displayFontSize}
                        onChange={e => {
                          const next = e.target.value === '' ? 0 : Number(e.target.value);
                          setSettings(s => ({ ...s, ui_font_size: Number.isNaN(next) ? 0 : next }));
                        }}
                        placeholder="0 = default"
                      />
                      <button type="button" className="secondary mini-btn" onClick={() => setSettings(s => ({ ...s, ui_font_size: 0 }))}>Use default</button>
                    </div>
                    <input
                      className="settings-range"
                      type="range"
                      min="11"
                      max="18"
                      value={displayFontSize}
                      onChange={e => setSettings(s => ({ ...s, ui_font_size: Number(e.target.value) }))}
                    />
                    <div className="form-help-text compact">
                      Drag for quick sizing. {isFontSizeDefault ? `Using default ${defaultFontSize}px.` : "0 keeps app default."}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div className="settings-card">
              <div className="settings-card-title">Text Colors</div>
              <div className="settings-card-body settings-fields cols-3">
                <div className="form-group">
                  <label>Primary</label>
                  <div className="settings-inline">
                    <input
                      type="color"
                      className="settings-color"
                      value={settings.ui_text_primary || settings.ui_font_color || '#e6edf6'}
                      onChange={e => setSettings(s => ({ ...s, ui_text_primary: e.target.value }))}
                    />
                    <input
                      value={settings.ui_text_primary}
                      onChange={e => setSettings(s => ({ ...s, ui_text_primary: e.target.value }))}
                      placeholder="#e6edf6"
                    />
                    <button type="button" className="secondary mini-btn" onClick={() => setSettings(s => ({ ...s, ui_text_primary: '' }))}>Clear</button>
                  </div>
                </div>
                <div className="form-group">
                  <label>Muted</label>
                  <div className="settings-inline">
                    <input
                      type="color"
                      className="settings-color"
                      value={settings.ui_text_muted || '#94a3b8'}
                      onChange={e => setSettings(s => ({ ...s, ui_text_muted: e.target.value }))}
                    />
                    <input
                      value={settings.ui_text_muted}
                      onChange={e => setSettings(s => ({ ...s, ui_text_muted: e.target.value }))}
                      placeholder="#94a3b8"
                    />
                    <button type="button" className="secondary mini-btn" onClick={() => setSettings(s => ({ ...s, ui_text_muted: '' }))}>Clear</button>
                  </div>
                </div>
                <div className="form-group">
                  <label>Accent</label>
                  <div className="settings-inline">
                    <input
                      type="color"
                      className="settings-color"
                      value={settings.ui_text_accent || '#4f8cff'}
                      onChange={e => setSettings(s => ({ ...s, ui_text_accent: e.target.value }))}
                    />
                    <input
                      value={settings.ui_text_accent}
                      onChange={e => setSettings(s => ({ ...s, ui_text_accent: e.target.value }))}
                      placeholder="#4f8cff"
                    />
                    <button type="button" className="secondary mini-btn" onClick={() => setSettings(s => ({ ...s, ui_text_accent: '' }))}>Clear</button>
                  </div>
                </div>
              </div>
              <div className="form-help-text compact">Primary maps to main text, muted to secondary labels, accent to highlights.</div>
            </div>

            <div className="modal-footer settings-footer">
              <button type="button" className="secondary" onClick={onClose}>Close</button>
              <button
                type="button"
                className="secondary"
                onClick={() => setSettings(s => ({
                  ...s,
                  theme_variant: '',
                  ui_theme_custom: '',
                  ui_font_family: '',
                  ui_font_size: 0,
                  ui_font_color: '',
                  ui_text_primary: '',
                  ui_text_muted: '',
                  ui_text_accent: '',
                }))}
              >
                Reset Theme
              </button>
              <button type="button" onClick={saveTheme} disabled={loading}>
                Save Theme
              </button>
            </div>
          </div>
        )}

      </div>
    </div>
  );
}
