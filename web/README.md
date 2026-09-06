# web

Frontend React + TypeScript (Vite) di Sanitas. Scaffold minimo: una pagina che elenca i turni chiamando il servizio `shifts` (vedi [../services/shifts](../services/shifts)).

```bash
npm install
npm run dev      # dev server
npm run build     # build di produzione
npm run lint      # oxlint
```

Configurazione via `.env` (vedi `.env.example`): `VITE_SHIFTS_API_URL` punta al servizio `shifts`.
