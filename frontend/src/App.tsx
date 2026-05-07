import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { SessionGate } from './components/SessionGate'
import { MapPage } from './pages/MapPage'
import { AdminLoginPage } from './pages/AdminLogin'
import { AdminPage } from './pages/Admin'
import { ContestantLoginPage } from './pages/ContestantLogin'
import { ContestantPage } from './pages/Contestant'

export default function App() {
    return (
        <BrowserRouter>
            <Routes>
                <Route path="/" element={<MapPage />} />
                <Route path="/admin-login" element={<AdminLoginPage />} />
                <Route
                    path="/admin"
                    element={
                        <SessionGate sessionEndpoint="/api/admin/session" loginPath="/admin-login">
                            <AdminPage />
                        </SessionGate>
                    }
                />
                <Route path="/contestant-login" element={<ContestantLoginPage />} />
                <Route
                    path="/contestant"
                    element={
                        <SessionGate sessionEndpoint="/api/contestant/session" loginPath="/contestant-login">
                            <ContestantPage />
                        </SessionGate>
                    }
                />
                <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
        </BrowserRouter>
    )
}
