import { Link, Outlet, useNavigate } from '@tanstack/react-router'
import type { LinkProps } from '@tanstack/react-router'
import { Building2, ExternalLink, FileClock, KeyRound, LayoutDashboard, Layers, LogOut, Shield, ShieldAlert, UserCircle, Users } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { auth, logout } from '../lib/api'

type NavItem = { to: LinkProps['to']; label: string; icon: LucideIcon; exact?: boolean }

const navItems: NavItem[] = [
    { to: '/', label: 'Panel', icon: LayoutDashboard, exact: true },
    { to: '/users', label: 'Kullanıcılar', icon: Users },
    { to: '/departments', label: 'Departmanlar', icon: Building2 },
    { to: '/groups', label: 'Gruplar', icon: Layers },
    { to: '/roles', label: 'Roller', icon: Shield },
    { to: '/clients', label: 'Clientlar', icon: KeyRound },
    { to: '/policies', label: 'Politikalar', icon: ShieldAlert },
    { to: '/audit', label: 'Denetim', icon: FileClock },
]

export function AppLayout() {
    const navigate = useNavigate()
    const user = auth.user
    const isAdmin = user?.roles.includes('admin') ?? false
    const visibleNavItems = isAdmin ? navItems : navItems.filter((item) => item.to === '/')

    const handleLogout = async () => {
        await logout()
        navigate({ to: '/login' })
    }

    return (
        <div className="app-container">
            <header className="top-nav">
                <Link to="/" className="brand-section">
                    <div className="logo-tag"><KeyRound size={14} />gOpenID</div>
                    <div className="brand-title">Kimlik Sunucusu<span>Yönetim</span></div>
                </Link>

                <nav className="nav-tabs-wrapper">
                    {visibleNavItems.map((item) => (
                        <Link
                            key={item.to}
                            to={item.to}
                            activeOptions={{ exact: item.exact ?? false }}
                            className="product-tab"
                            activeProps={{ className: 'product-tab active' }}
                        >
                            <item.icon size={14} />
                            {item.label}
                        </Link>
                    ))}
                </nav>

                <div className="nav-actions">
                    <a className="nav-link-meta" href="/.well-known/openid-configuration" target="_blank" rel="noreferrer">
                        <ExternalLink size={14} />
                        OIDC keşfi
                    </a>
                    <Link to="/profile" className="nav-user" title="Profilim">
                        <UserCircle size={16} />
                        <span>{user?.name ?? 'Profil'}</span>
                    </Link>
                    <button className="btn-signout" onClick={handleLogout}>
                        <LogOut size={14} />
                        Çıkış
                    </button>
                </div>
            </header>

            <main className="workspace">
                <Outlet />
            </main>
        </div>
    )
}
