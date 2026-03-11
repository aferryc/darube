import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import React from 'react'
import { ConnectionModal } from './ConnectionModal'

function Wrapper() {
  const [formData, setFormData] = React.useState({
    connection_name: 'Test',
    db_type: 'postgres',
    host: 'localhost',
    port: 5432,
    dbname: '',
    file_path: '',
    user: 'u',
    password: '',
    enable_ssl: false,
    ca_cert_path: '',
    client_cert_path: '',
    client_key_path: '',
    folder_id: '',
    is_cluster: false,
  })

  return (
    <ConnectionModal
      show
      editingId={null}
      formData={formData}
      setFormData={setFormData}
      folders={[]}
      onSubmit={(e) => e.preventDefault()}
      onTest={(e) => e.preventDefault()}
      onClose={() => {}}
    />
  )
}

function WrapperWithFolders() {
  const [formData, setFormData] = React.useState({
    connection_name: 'Test',
    db_type: 'sqlite',
    host: '',
    port: 0,
    dbname: '',
    file_path: '',
    user: '',
    password: '',
    enable_ssl: false,
    ca_cert_path: '',
    client_cert_path: '',
    client_key_path: '',
    folder_id: '',
    is_cluster: false,
  })

  return (
    <ConnectionModal
      show
      editingId={null}
      formData={formData}
      setFormData={setFormData}
      folders={[{ id: 'f1', name: 'Ops' }]}
      onSubmit={(e) => e.preventDefault()}
      onTest={(e) => e.preventDefault()}
      onClose={() => {}}
    />
  )
}

describe('ConnectionModal (NoSQL types)', () => {
  it('shows NoSQL types and sets default ports + fields', async () => {
    const user = userEvent.setup()
    render(<Wrapper />)

    await user.click(screen.getByText('NoSQL'))

    const typeSelect = screen.getByRole('combobox')
    expect(typeSelect).toHaveDisplayValue('Redis')
    const portInput = screen.getByRole('spinbutton')

    // Switch to MongoDB -> default port should be 27017, and show Database field.
    await user.selectOptions(typeSelect, 'mongodb')
    expect(portInput).toHaveValue(27017)
    expect(screen.getByPlaceholderText('app')).toBeInTheDocument()
    expect(screen.queryByText('Enable Cluster Mode')).not.toBeInTheDocument()

    // Switch to Cassandra -> default port should be 9042, and show Keyspace field.
    await user.selectOptions(typeSelect, 'cassandra')
    expect(portInput).toHaveValue(9042)
    expect(screen.getByPlaceholderText('keyspace')).toBeInTheDocument()

    // Switch to Elasticsearch -> default port 9200, and show Index field.
    await user.selectOptions(typeSelect, 'elasticsearch')
    expect(portInput).toHaveValue(9200)
    expect(screen.getByPlaceholderText('logs-*')).toBeInTheDocument()

    // Switch back to Redis -> cluster toggle shows.
    await user.selectOptions(typeSelect, 'redis')
    expect(portInput).toHaveValue(6379)
    expect(screen.getByText('Enable Cluster Mode')).toBeInTheDocument()
  })
})

describe('ConnectionModal (SQL types)', () => {
  it('switches between sqlite and oracle and shows the correct fields', async () => {
    const user = userEvent.setup()
    render(<Wrapper />)

    // SQL is default: pick SQLite
    const typeSelect = screen.getByDisplayValue('PostgreSQL')
    await user.selectOptions(typeSelect, 'sqlite')

    expect(screen.getByText('Database File')).toBeInTheDocument()
    expect(screen.getByPlaceholderText(/db\.sqlite/)).toBeInTheDocument()
    expect(screen.queryByText('SSL')).not.toBeInTheDocument()

    // Switch to Oracle
    await user.selectOptions(typeSelect, 'oracle')
    expect(screen.getByText('Service Name')).toBeInTheDocument()
    const svc = screen.getByPlaceholderText('orclpdb1')
    expect(svc).toBeRequired()
  })

  it('shows SSL fields when enabled (non-sqlite)', async () => {
    const user = userEvent.setup()
    render(<Wrapper />)

    // Ensure not sqlite so SSL tab exists
    await user.click(screen.getByRole('button', { name: 'SSL' }))
    await user.click(screen.getByLabelText('Enable SSL Configuration'))
    expect(screen.getByText('CA Certificate (PEM/CRT)')).toBeInTheDocument()
    expect(screen.getByText('Client Certificate')).toBeInTheDocument()
    expect(screen.getByText('Client Key')).toBeInTheDocument()
  })

  it('renders folder selector and browse button updates sqlite file_path when electron is available', async () => {
    const user = userEvent.setup()

    const invoke = vi.fn().mockResolvedValue('/tmp/test.sqlite')
    const prevRequire = window.require
    window.require = () => ({ ipcRenderer: { invoke } })

    render(<WrapperWithFolders />)

    // Folder selector should be visible (folders.length > 0).
    const selects = screen.getAllByRole('combobox')
    const folderSelect = selects.find(s => Array.from(s.options || []).some(o => o.textContent === 'Ops'))
    expect(folderSelect).toBeTruthy()
    await user.selectOptions(folderSelect, 'f1')

    // Browse button should update file path.
    const browse = screen.getByRole('button', { name: 'Browse...' })
    await user.click(browse)
    expect(invoke).toHaveBeenCalled()
    expect(screen.getByPlaceholderText(/db\.sqlite/)).toHaveValue('/tmp/test.sqlite')

    window.require = prevRequire
  })
})
