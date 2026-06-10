import { lazy, Suspense } from 'react'
import type { ComponentType } from 'react'
import { createRootRoute, createRoute, createRouter, redirect } from '@tanstack/react-router'
import { auth } from './lib/api'
import { AppLayout } from './components/AppLayout'

const LoginPage = lazy(() => import('./pages/LoginPage').then((m) => ({ default: m.LoginPage })))
const DashboardPage = lazy(() => import('./pages/DashboardPage').then((m) => ({ default: m.DashboardPage })))
const UsersPage = lazy(() => import('./pages/UsersPage').then((m) => ({ default: m.UsersPage })))
const DepartmentsPage = lazy(() => import('./pages/DepartmentsPage').then((m) => ({ default: m.DepartmentsPage })))
const GroupsPage = lazy(() => import('./pages/GroupsPage').then((m) => ({ default: m.GroupsPage })))
const RolesPage = lazy(() => import('./pages/RolesPage').then((m) => ({ default: m.RolesPage })))
const ClientsPage = lazy(() => import('./pages/ClientsPage').then((m) => ({ default: m.ClientsPage })))
const PoliciesPage = lazy(() => import('./pages/PoliciesPage').then((m) => ({ default: m.PoliciesPage })))
const AuditPage = lazy(() => import('./pages/AuditPage').then((m) => ({ default: m.AuditPage })))
const ProfilePage = lazy(() => import('./pages/ProfilePage').then((m) => ({ default: m.ProfilePage })))

const rootRoute = createRootRoute()

// Public login route. Redirects to the dashboard when already authenticated.
const loginRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/login',
    beforeLoad: () => {
        if (auth.isTokenValid()) throw redirect({ to: '/' })
    },
    component: withSuspense(LoginPage),
})

// Authenticated shell. Guards every child route and renders the navigation.
const appLayoutRoute = createRoute({
    getParentRoute: () => rootRoute,
    id: 'app',
    beforeLoad: ({ location }) => {
        if (!auth.isTokenValid()) {
            auth.clear()
            throw redirect({ to: '/login', search: { redirect: location.href } })
        }
    },
    component: AppLayout,
})

const dashboardRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/', component: withSuspense(DashboardPage) })
const usersRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/users', component: withSuspense(UsersPage) })
const departmentsRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/departments', component: withSuspense(DepartmentsPage) })
const groupsRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/groups', component: withSuspense(GroupsPage) })
const rolesRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/roles', component: withSuspense(RolesPage) })
const clientsRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/clients', component: withSuspense(ClientsPage) })
const policiesRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/policies', component: withSuspense(PoliciesPage) })
const auditRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/audit', component: withSuspense(AuditPage) })
const profileRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/profile', component: withSuspense(ProfilePage) })

const routeTree = rootRoute.addChildren([
    loginRoute,
    appLayoutRoute.addChildren([
        dashboardRoute,
        usersRoute,
        departmentsRoute,
        groupsRoute,
        rolesRoute,
        clientsRoute,
        policiesRoute,
        auditRoute,
        profileRoute,
    ]),
])

export const router = createRouter({ routeTree })

function withSuspense(Component: ComponentType) {
    return function LazyRoute() {
        return (
            <Suspense fallback={<div className="state-empty"><strong>Sayfa yükleniyor</strong><p>Modül hazırlanıyor.</p></div>}>
                <Component />
            </Suspense>
        )
    }
}

declare module '@tanstack/react-router' {
    interface Register {
        router: typeof router
    }
}
