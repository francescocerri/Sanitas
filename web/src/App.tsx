import { useEffect, useState } from 'react'

type Turno = {
  id: string
  volontario_id: string
  data: string
  ora_inizio: string
  ora_fine: string
  stato: string
}

const API_BASE_URL = import.meta.env.VITE_TURNI_API_URL ?? 'http://localhost:8080'

function App() {
  const [turni, setTurni] = useState<Turno[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch(`${API_BASE_URL}/v1/turni`)
      .then((res) => {
        if (!res.ok) throw new Error(`API turni: ${res.status}`)
        return res.json() as Promise<Turno[]>
      })
      .then(setTurni)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
  }, [])

  return (
    <main>
      <h1>Turni</h1>
      {error && <p role="alert">Errore: {error}</p>}
      {!error && turni.length === 0 && <p>Nessun turno pianificato.</p>}
      {turni.length > 0 && (
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
            {turni.map((t) => (
              <tr key={t.id}>
                <td>{t.data}</td>
                <td>
                  {t.ora_inizio}–{t.ora_fine}
                </td>
                <td>{t.volontario_id}</td>
                <td>{t.stato}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </main>
  )
}

export default App
