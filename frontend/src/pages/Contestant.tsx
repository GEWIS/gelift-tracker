import { useEffect, useState } from 'react'

type ContestantLink = {
    label: string
    inline: string
}

export function ContestantPage() {
    const [rows, setRows] = useState<ContestantLink[] | null>(null)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        fetch('/api/contestant/links', { credentials: 'include' })
            .then(async (res) => {
                const data = (await res.json().catch(() => ({}))) as {
                    contestants?: ContestantLink[]
                    error?: string
                }
                if (!res.ok) {
                    setError(typeof data.error === 'string' ? data.error : 'Could not load contestant links')
                    setRows([])
                    return
                }
                setRows(Array.isArray(data.contestants) ? data.contestants : [])
                if (typeof data.error === 'string' && data.error !== '') {
                    setError(data.error)
                }
            })
            .catch(() => {
                setError('Could not load contestant links')
                setRows([])
            })
    }, [])

    if (rows === null) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-50 text-slate-600">
                Loading…
            </div>
        )
    }

    return (
        <div className="min-h-screen flex flex-col items-center bg-slate-50 p-6 gap-6">
            <div className="text-center">
                <h1 className="text-2xl font-semibold text-slate-800 m-0">Contestant setup</h1>
                <p className="text-slate-600 m-0 mt-2 max-w-lg">
                    Tap the button with your name on it to load the configuration into Owntracks.
                </p>
            </div>
            {error && (
                <p className="text-sm text-amber-800 bg-amber-50 border border-amber-200 rounded px-4 py-2 m-0" role="status">
                    {error}
                </p>
            )}
            <div className="flex flex-col gap-3 w-full max-w-md">
                {rows.length === 0 ? (
                    <p className="text-slate-600 text-center m-0">No contestant links are configured.</p>
                ) : (
                    rows.map((c) => (
                        <a
                            key={`${c.label}:${c.inline.slice(0, 24)}`}
                            href={`owntracks:///config?inline=${encodeURIComponent(c.inline)}`}
                            className="inline-flex items-center justify-center rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-medium px-4 py-3 no-underline shadow-sm transition-colors"
                        >
                            {c.label}
                        </a>
                    ))
                )}
            </div>
        </div>
    )
}
