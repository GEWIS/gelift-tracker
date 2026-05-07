import { PasswordLoginPage } from '../components/PasswordLoginPage'

export function ContestantLoginPage() {
    return (
        <PasswordLoginPage
            title="Contestant login"
            loginEndpoint="/api/contestant/login"
            redirectTo="/contestant"
            passwordFieldId="contestant-password"
        />
    )
}
