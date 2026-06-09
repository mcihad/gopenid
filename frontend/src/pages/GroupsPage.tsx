import { PageHeader } from '../components/ui'
import { ResourcePanel } from '../components/ResourcePanel'

export function GroupsPage() {
    return (
        <div className="directory-section">
            <PageHeader eyebrow="Organizasyon" title="Kullanıcı grupları" />
            <p className="section-note">Kullanıcılar birden fazla gruba dahil olabilir. Gruplara giriş politikaları atanabilir.</p>
            <ResourcePanel kind="groups" title="Gruplar" single="Grup" embedded />
        </div>
    )
}
