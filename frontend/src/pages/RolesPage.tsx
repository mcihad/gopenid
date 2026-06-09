import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { api } from '../lib/api'
import type { Client, ClientRole } from '../lib/types'
import { Actions, Alert, ControlBar, EmptySearch, Field, Modal, ModalFooter, PageHeader, SearchInput, TableState, Tag } from '../components/ui'
import { lower } from '../lib/format'
import { ResourcePanel } from '../components/ResourcePanel'

type RoleTab = 'global' | 'client'

export function RolesPage() {
  const [roleTab, setRoleTab] = useState<RoleTab>('global')
  return (
    <div className="directory-section">
      <PageHeader eyebrow="Erişim kontrolü" title="Roller" />
      <div className="section-tabs">
        <button className={`section-tab ${roleTab === 'global' ? 'active' : ''}`} onClick={() => setRoleTab('global')}>Sistem rolleri</button>
        <button className={`section-tab ${roleTab === 'client' ? 'active' : ''}`} onClick={() => setRoleTab('client')}>Client rolleri</button>
      </div>
      {roleTab === 'global' ? <ResourcePanel kind="roles" title="Sistem rolleri" single="Rol" embedded /> : <ClientRolesPanel />}
    </div>
  )
}

function ClientRolesPanel() {
  const qc = useQueryClient()
  const clients = useQuery({ queryKey: ['clients'], queryFn: api.clients.list })
  const [selectedClientID, setSelectedClientID] = useState<number | 'all'>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [edit, setEdit] = useState<{ client: Client; role: ClientRole } | null>(null)
  const [draft, setDraft] = useState({ name: '', description: '', clientID: 0 })

  const roles = useMemo(() => {
    const q = lower(searchQuery)
    return (clients.data ?? [])
      .filter((client) => selectedClientID === 'all' || client.ID === selectedClientID)
      .flatMap((client) => (client.roles ?? []).map((role) => ({ client, role })))
      .filter(({ client, role }) => lower(role.name).includes(q) || lower(role.description).includes(q) || lower(client.name).includes(q) || lower(client.clientId).includes(q))
  }, [clients.data, selectedClientID, searchQuery])

  const save = useMutation({
    mutationFn: () => {
      if (edit) return api.clients.roles.update(edit.client.ID, edit.role.ID, { name: draft.name, description: draft.description })
      return api.clients.roles.create(draft.clientID, { name: draft.name, description: draft.description })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['clients'] })
      closeModal()
    },
  })
  const remove = useMutation({
    mutationFn: ({ clientID, roleID }: { clientID: number; roleID: number }) => api.clients.roles.remove(clientID, roleID),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['clients'] })
      qc.invalidateQueries({ queryKey: ['users'] })
    },
  })

  const openCreateModal = () => {
    const firstClientID = selectedClientID === 'all' ? clients.data?.[0]?.ID ?? 0 : selectedClientID
    setEdit(null)
    setDraft({ name: '', description: '', clientID: firstClientID })
    save.reset()
    setIsModalOpen(true)
  }
  const openEditModal = (client: Client, role: ClientRole) => {
    setEdit({ client, role })
    setDraft({ name: role.name, description: role.description, clientID: client.ID })
    save.reset()
    setIsModalOpen(true)
  }
  const closeModal = () => {
    setIsModalOpen(false)
    setEdit(null)
    setDraft({ name: '', description: '', clientID: 0 })
  }

  return (
    <div>
      <ControlBar onRefresh={() => clients.refetch()} action={<button className="btn-primary" onClick={openCreateModal} disabled={!clients.data?.length}><Plus size={14} />Client rolü ekle</button>}>
        <div className="role-filters">
          <SearchInput value={searchQuery} onChange={setSearchQuery} placeholder="Client rolü veya client ara..." />
          <select value={selectedClientID} onChange={(event) => setSelectedClientID(event.target.value === 'all' ? 'all' : Number(event.target.value))}>
            <option value="all">Tüm clientlar</option>
            {clients.data?.map((client) => <option key={client.ID} value={client.ID}>{client.name} ({client.clientId})</option>)}
          </select>
        </div>
      </ControlBar>

      <TableState query={clients} />
      {clients.data && clients.data.length > 0 && roles.length === 0 && <EmptySearch text="Bu filtrelerle eşleşen client rolü yok." />}

      {roles.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr><th>Rol</th><th>Client</th><th>Açıklama</th><th></th></tr>
            </thead>
            <tbody>
              {roles.map(({ client, role }) => (
                <tr key={`${client.ID}-${role.ID}`} onClick={() => openEditModal(client, role)}>
                  <td><strong>{role.name}</strong></td>
                  <td><Tag primary>{client.name}</Tag><span className="mono-text">{client.clientId}</span></td>
                  <td>{role.description || 'Açıklama girilmemiş.'}</td>
                  <td>
                    <Actions
                      onEdit={(event) => { event.stopPropagation(); openEditModal(client, role) }}
                      onDelete={(event) => { event.stopPropagation(); if (confirm(`${role.name} client rolü silinsin mi?`)) remove.mutate({ clientID: client.ID, roleID: role.ID }) }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal isOpen={isModalOpen} onClose={closeModal} title={edit ? `Client rolü düzenle: ${edit.role.name}` : 'Yeni client rolü'}>
        <form onSubmit={(event) => { event.preventDefault(); save.mutate() }}>
          <div className="modal-body">
            <div className="modal-form">
              {save.error && <Alert type="error" message={save.error.message} />}
              <Field label="Client">
                <select value={draft.clientID} onChange={(event) => setDraft({ ...draft, clientID: Number(event.target.value) })} disabled={Boolean(edit)} required>
                  <option value={0}>Client seçin</option>
                  {clients.data?.map((client) => <option key={client.ID} value={client.ID}>{client.name} ({client.clientId})</option>)}
                </select>
              </Field>
              <Field label="Rol adı">
                <input placeholder="örn. reader" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} required />
              </Field>
              <Field label="Açıklama">
                <textarea placeholder="Rolün client içindeki yetki kapsamı..." value={draft.description} onChange={(event) => setDraft({ ...draft, description: event.target.value })} />
              </Field>
            </div>
          </div>
          <ModalFooter onCancel={closeModal} pending={save.isPending} submitText={edit ? 'Değişiklikleri kaydet' : 'Oluştur'} />
        </form>
      </Modal>
    </div>
  )
}
