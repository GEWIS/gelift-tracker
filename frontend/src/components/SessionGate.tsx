import { useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'

type SessionPayload = {
    configured?: boolean
    authenticated?: boolean
}

type Props = {
    sessionEndpoint: string
    loginPath: string
    children: React.ReactNode
}

export function SessionGate({ sessionEndpoint, loginPath, children }: Props) {
    const [state, setState] = useState<'loading' | 'allow' | 'deny'>('loading')

    useEffect(() => {
        fetch(sessionEndpoint, { credentials: 'include' })
            .then(async (res) => {
                const data = (await res.json().catch(() => ({}))) as SessionPayload
                if (data.configured === false) {
                    setState('allow')
                    return
                }
                setState(res.ok && data.authenticated ? 'allow' : 'deny')
            })
            .catch(() => setState('deny'))
    }, [sessionEndpoint])

    if (state === 'loading') {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-50 text-slate-600">
                Loading…
            </div>
        )
    }
    if (state === 'deny') {
        return <Navigate to={loginPath} replace />
    }
    return children
}
