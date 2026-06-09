import { PageHeader } from '../components/ui'
import { ResourcePanel } from '../components/ResourcePanel'

export function DepartmentsPage() {
    return (
        <div className="directory-section">
            <PageHeader eyebrow="Organizasyon" title="Departmanlar" />
            <ResourcePanel kind="departments" title="Departmanlar" single="Departman" embedded />
        </div>
    )
}
