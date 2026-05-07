import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from 'primereact/button'
import { Password } from 'primereact/password'

type Props = {
    title: string
    loginEndpoint: string
    redirectTo: string
    passwordFieldId: string
}

export function PasswordLoginPage({ title, loginEndpoint, redirectTo, passwordFieldId }: Props) {
    const [password, setPassword] = useState('')
    const [error, setError] = useState<string | null>(null)
    const [loading, setLoading] = useState(false)
    const navigate = useNavigate()

    async function onSubmit(e: React.FormEvent) {
        e.preventDefault()
        setError(null)
        setLoading(true)
        try {
            const res = await fetch(loginEndpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify({ password }),
            })
            const data = await res.json().catch(() => ({}))
            if (!res.ok) {
                setError(typeof data.error === 'string' ? data.error : 'Login failed')
                return
            }
            navigate(redirectTo, { replace: true })
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-slate-100 p-4">
            <form
                onSubmit={onSubmit}
                className="w-full max-w-sm bg-white rounded-lg shadow-md p-8 flex flex-col gap-4 text-left"
            >
                <h1 className="text-xl font-semibold text-slate-800 m-0">{title}</h1>
                <div className="flex flex-col gap-2">
                    <label htmlFor={passwordFieldId} className="text-sm text-slate-600">
                        Password
                    </label>
                    <Password
                        id={passwordFieldId}
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        feedback={false}
                        toggleMask
                        className="w-full"
                        inputClassName="w-full"
                        disabled={loading}
                        autoComplete="current-password"
                    />
                </div>
                {error && (
                    <p className="text-sm text-red-600 m-0" role="alert">
                        {error}
                    </p>
                )}
                <Button type="submit" label={loading ? 'Signing in…' : 'Sign in'} loading={loading} disabled={loading} />
            </form>
        </div>
    )
}
