import { PasswordLoginPage } from '../components/PasswordLoginPage'

export function AdminLoginPage() {
    return (
        <PasswordLoginPage
            title="Admin login"
            loginEndpoint="/api/admin/login"
            redirectTo="/admin"
            passwordFieldId="admin-password"
        />
    )
}
