import { createRootRoute, createRoute, createRouter, redirect } from '@tanstack/react-router'
import { auth } from './lib/api'
import { AppLayout } from './components/AppLayout'
import { LoginPage } from './pages/LoginPage'
import { DashboardPage } from './pages/DashboardPage'
import { UsersPage } from './pages/UsersPage'
import { DepartmentsPage } from './pages/DepartmentsPage'
import { GroupsPage } from './pages/GroupsPage'
import { RolesPage } from './pages/RolesPage'
import { ClientsPage } from './pages/ClientsPage'
import { PoliciesPage } from './pages/PoliciesPage'
import { AuditPage } from './pages/AuditPage'
import { ProfilePage } from './pages/ProfilePage'

const rootRoute = createRootRoute()

// Public login route. Redirects to the dashboard when already authenticated.
const loginRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/login',
    beforeLoad: () => {
        if (auth.isTokenValid()) throw redirect({ to: '/' })
    },
    component: LoginPage,
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

const dashboardRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/', component: DashboardPage })
const usersRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/users', component: UsersPage })
const departmentsRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/departments', component: DepartmentsPage })
const groupsRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/groups', component: GroupsPage })
const rolesRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/roles', component: RolesPage })
const clientsRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/clients', component: ClientsPage })
const policiesRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/policies', component: PoliciesPage })
const auditRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/audit', component: AuditPage })
const profileRoute = createRoute({ getParentRoute: () => appLayoutRoute, path: '/profile', component: ProfilePage })

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

declare module '@tanstack/react-router' {
    interface Register {
        router: typeof router
    }
}
