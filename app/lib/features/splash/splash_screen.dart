import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';

import '../../core/widgets/brand_mark.dart';

/// Mostrata solo per l'istante in cui l'app non sa ancora se c'è una
/// sessione da riprendere (`AuthStatus.unknown` — vedi `router.dart` e
/// `AuthController._bootstrap`). Dura tipicamente pochi decimi di secondo:
/// il tempo di un'unica chiamata a `POST /v1/refresh` se c'è un refresh
/// token salvato, altrimenti è quasi istantanea.
class SplashScreen extends StatelessWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        // Un leggero effetto "respiro" (scala su e giù, in loop) al posto
        // di un semplice spinner: comunica "sto caricando" senza sembrare
        // statico. `onPlay: (c) => c.repeat(...)` dice all'animazione di
        // ricominciare da capo ogni volta che finisce, all'infinito.
        child: const BrandMark(size: 88)
            .animate(onPlay: (controller) => controller.repeat(reverse: true))
            .scale(
              begin: const Offset(0.92, 0.92),
              end: const Offset(1.0, 1.0),
              duration: 900.ms,
              curve: Curves.easeInOut,
            ),
      ),
    );
  }
}
