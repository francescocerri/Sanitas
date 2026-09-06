import { useEffect, useState } from 'react'

type Shift = {
  id: string
  volunteer_id: string
  date: string
  start_time: string
  end_time: string
  status: string
}

const API_BASE_URL = import.meta.env.VITE_SHIFTS_API_URL ?? 'http://localhost:8080'

function App() {
  const [shifts, setShifts] = useState<Shift[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch(`${API_BASE_URL}/v1/shifts`)
      .then((res) => {
        if (!res.ok) throw new Error(`API turni: ${res.status}`)
        return res.json() as Promise<Shift[]>
      })
      .then(setShifts)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
  }, [])

  return (
    <main>
      <h1>Turni</h1>
      {error && <p role="alert">Errore: {error}</p>}
      {!error && shifts.length === 0 && <p>Nessun turno pianificato.</p>}
      {shifts.length > 0 && (
        <table>
          <thead>
            <tr>
              <th>Data</th>
              <th>Orario</th>
              <th>Volontario</th>
              <th>Stato</th>
            </tr>
          </thead>
          <tbody>
            {shifts.map((s) => (
              <tr key={s.id}>
                <td>{s.date}</td>
                <td>
                  {s.start_time}–{s.end_time}
                </td>
                <td>{s.volunteer_id}</td>
                <td>{s.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </main>
  )
}

export default App
