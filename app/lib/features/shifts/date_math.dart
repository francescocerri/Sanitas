/// Solo la parte di data (mezzanotte locale) di [d] — usata ovunque nelle
/// viste Turni per confrontare date ignorando l'orario (le occorrenze del
/// backend arrivano come `YYYY-MM-DDT00:00:00Z`).
DateTime dateOnly(DateTime d) => DateTime(d.year, d.month, d.day);

/// Somma [n] giorni di calendario a [d] (può essere negativo). Volutamente
/// NON `d.add(Duration(days: n))`: l'aritmetica su `Duration` opera in ore
/// fisse e può sbagliare giorno attraversando il cambio ora legale — la
/// costruzione componente-per-componente (che `DateTime` normalizza da
/// solo, anche oltre i confini di mese/anno) non ha questo problema.
DateTime addDays(DateTime d, int n) => DateTime(d.year, d.month, d.day + n);
