# web

Frontend React + TypeScript (Vite) di Sanitas. Scaffold minimo: una pagina che elenca i turni chiamando il servizio `turni` (vedi [../services/turni](../services/turni)).

```bash
npm install
npm run dev      # dev server
npm run build     # build di produzione
npm run lint      # oxlint
```

Configurazione via `.env` (vedi `.env.example`): `VITE_TURNI_API_URL` punta al servizio `turni`.
