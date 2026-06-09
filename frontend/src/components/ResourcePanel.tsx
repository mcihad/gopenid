import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { api } from '../lib/api'
import type { Resource } from '../lib/types'
import { Actions, Alert, ControlBar, EmptySearch, Field, Modal, ModalFooter, SearchInput, TableState } from './ui'
import { lower } from '../lib/format'

type Kind = 'departments' | 'roles' | 'groups'

// ResourcePanel is a generic CRUD table for simple name/description records
// (departments, system roles and user groups).
export function ResourcePanel({ kind, title, single, embedded = false }: { kind: Kind; title: string; single: string; embedded?: boolean }) {
  const qc = useQueryClient()
  const resource = api[kind]
  const query = useQuery({ queryKey: [kind], queryFn: resource.list })
  const users = useQuery({ queryKey: ['users'], queryFn: api.users.list })
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [edit, setEdit] = useState<Resource | null>(null)
  const [draft, setDraft] = useState({ name: '', description: '' })
  const [searchQuery, setSearchQuery] = useState('')

  const save = useMutation({
    mutationFn: () => (edit ? resource.update(edit.ID, draft) : resource.create(draft)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [kind] })
      closeModal()
    },
  })
  const remove = useMutation({
    mutationFn: (id: number) => resource.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: [kind] }),
  })

  const filteredData = useMemo(() => {
    const q = lower(searchQuery)
    return (query.data ?? []).filter((row) => lower(row.name).includes(q) || lower(row.description).includes(q))
  }, [query.data, searchQuery])

  const openCreateModal = () => {
    setEdit(null)
    setDraft({ name: '', description: '' })
    save.reset()
    setIsModalOpen(true)
  }
  const openEditModal = (row: Resource) => {
    setEdit(row)
    setDraft({ name: row.name, description: row.description })
    save.reset()
    setIsModalOpen(true)
  }
  const closeModal = () => {
    setIsModalOpen(false)
    setEdit(null)
    setDraft({ name: '', description: '' })
  }
  const getUserCount = (id: number) => {
    if (kind === 'departments') return users.data?.filter((user) => user.departmentId === id || user.departments?.some((d) => d.ID === id)).length ?? 0
    if (kind === 'groups') return users.data?.filter((user) => user.groups?.some((g) => g.ID === id)).length ?? 0
    return users.data?.filter((user) => user.roles?.some((role) => role.ID === id)).length ?? 0
  }

  return (
    <div className={embedded ? '' : 'directory-section'}>
      <ControlBar onRefresh={() => query.refetch()} action={<button className="btn-primary" onClick={openCreateModal}><Plus size={14} />{single} ekle</button>}>
        <SearchInput value={searchQuery} onChange={setSearchQuery} placeholder={`${title} içinde ara...`} />
      </ControlBar>

      <TableState query={query} />
      {query.data && query.data.length > 0 && filteredData.length === 0 && <EmptySearch text={`"${searchQuery}" için kayıt bulunamadı.`} />}

      {filteredData.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr><th>Ad</th><th>Açıklama</th><th>Atanan kullanıcı</th><th></th></tr>
            </thead>
            <tbody>
              {filteredData.map((row) => (
                <tr key={row.ID} onClick={() => openEditModal(row)}>
                  <td><strong>{row.name}</strong></td>
                  <td>{row.description || 'Açıklama girilmemiş.'}</td>
                  <td>{getUserCount(row.ID)}</td>
                  <td>
                    <Actions
                      onEdit={(event) => { event.stopPropagation(); openEditModal(row) }}
                      onDelete={(event) => { event.stopPropagation(); if (confirm(`${single} silinsin mi?`)) remove.mutate(row.ID) }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal isOpen={isModalOpen} onClose={closeModal} title={edit ? `${single} düzenle: ${edit.name}` : `Yeni ${single}`}>
        <form onSubmit={(event) => { event.preventDefault(); save.mutate() }}>
          <div className="modal-body">
            <div className="modal-form">
              {save.error && <Alert type="error" message={save.error.message} />}
              <Field label="Ad">
                <input placeholder={single === 'Rol' ? 'örn. admin' : 'örn. Mühendislik'} value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} required />
              </Field>
              <Field label="Açıklama">
                <textarea placeholder="Bu kaydın kullanım amacını yazın..." value={draft.description} onChange={(event) => setDraft({ ...draft, description: event.target.value })} />
              </Field>
            </div>
          </div>
          <ModalFooter onCancel={closeModal} pending={save.isPending} submitText={edit ? 'Değişiklikleri kaydet' : 'Oluştur'} />
        </form>
      </Modal>
    </div>
  )
}
