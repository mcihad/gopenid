import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CornerDownRight, Plus } from 'lucide-react'
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
  const [draft, setDraft] = useState<{ name: string; description: string; parentId?: number }>({ name: '', description: '', parentId: undefined })
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
    setDraft({ name: '', description: '', parentId: undefined })
    save.reset()
    setIsModalOpen(true)
  }
  const openEditModal = (row: Resource) => {
    setEdit(row)
    setDraft({ name: row.name, description: row.description, parentId: 'parentId' in row && row.parentId ? row.parentId : undefined })
    save.reset()
    setIsModalOpen(true)
  }
  const closeModal = () => {
    setIsModalOpen(false)
    setEdit(null)
    setDraft({ name: '', description: '', parentId: undefined })
  }
  const tableRows = kind === 'departments' ? departmentRows(filteredData) : filteredData.map((row) => ({ row, depth: 0 }))
  const parentOptions = kind === 'departments' ? (query.data ?? []).filter((row) => row.ID !== edit?.ID) : []
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

      {tableRows.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr><th>Ad</th><th>Açıklama</th><th>Atanan kullanıcı</th><th></th></tr>
            </thead>
            <tbody>
              {tableRows.map(({ row, depth }) => (
                <tr key={row.ID} onClick={() => openEditModal(row)}>
                  <td>
                    <div className="tree-cell" style={{ paddingLeft: depth * 24 }}>
                      {kind === 'departments' && depth > 0 && <CornerDownRight size={14} />}
                      <strong>{row.name}</strong>
                    </div>
                  </td>
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
              {kind === 'departments' && (
                <Field label="Üst departman">
                  <select value={draft.parentId ?? ''} onChange={(event) => setDraft({ ...draft, parentId: event.target.value ? Number(event.target.value) : undefined })}>
                    <option value="">Kök departman</option>
                    {parentOptions.map((row) => <option key={row.ID} value={row.ID}>{row.name}</option>)}
                  </select>
                </Field>
              )}
            </div>
          </div>
          <ModalFooter onCancel={closeModal} pending={save.isPending} submitText={edit ? 'Değişiklikleri kaydet' : 'Oluştur'} />
        </form>
      </Modal>
    </div>
  )
}

function departmentRows(rows: Resource[]) {
  const byParent = new Map<number | 'root', Resource[]>()
  rows.forEach((row) => {
    const parentId = 'parentId' in row && row.parentId ? row.parentId : 'root'
    byParent.set(parentId, [...(byParent.get(parentId) ?? []), row])
  })
  const out: Array<{ row: Resource; depth: number }> = []
  const visit = (parentId: number | 'root', depth: number) => {
    ;(byParent.get(parentId) ?? []).forEach((row) => {
      out.push({ row, depth })
      visit(row.ID, depth + 1)
    })
  }
  visit('root', 0)
  return out
}
